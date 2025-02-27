<template>
  <div class="chat-container">
    <div v-if="currentUser" class="profile-info">
      Logged in as: <strong>{{ currentUser.username }}</strong>
    </div>

    <!-- Display either selected user/group or the selection lists -->
    <div v-if="selectedUser || selectedGroup">
      <!-- Chatting with a User -->
      <div v-if="selectedUser">
        <h2>Chatting with {{ selectedUser.username }}</h2>
        <div class="chat-messages" ref="messagesContainer" @scroll="handleScroll">
          <div v-if="loadingMore" class="loading-indicator">Loading...</div>
          <ChatMessages :messages="filteredMessages"/>
        </div>
        <ChatInput :receiverID="selectedUser.id" :groupID="null"/>
      </div>
      <!-- Chatting with a group -->
      <div v-else-if="selectedGroup">
        <div class="group-header">
          <h2>{{ selectedGroup.name }}</h2>
          <div class="group-code-container">
        <span class="group-code" title="Share this code to invite others">
          Group Code: {{ selectedGroup.code }}
        </span>
            <button class="copy-button" @click.stop="copyCode(selectedGroup.code)"
                    title="Copy to clipboard">
              Copy Code
            </button>
            <button class="leave-button" @click="confirmLeaveGroup"
                    title="Leave this group">
              Leave Group
            </button>
          </div>
          <span v-if="copyMessage" class="copy-message">{{ copyMessage }}</span>
        </div>
        <!-- Add section to display group members -->
        <div class="group-members">
          <h3>Members</h3>
          <ul>
            <li v-for="member in groupMembers" :key="member.id">
              {{ member.username }}
              <span :class="{ 'online-dot': isUserOnline(member.id) }"></span>
            </li>
          </ul>
        </div>
        <!--Add loading -->
        <div v-if="loadingMembers" class="loading-members">
          Loading members...
        </div>
        <div class="chat-messages" ref="messagesContainer" @scroll="handleScroll">
          <div v-if="loadingMore" class="loading-indicator">Loading...</div>
          <ChatMessages :messages="filteredMessages"/>
        </div>
        <ChatInput :groupID="selectedGroup.id" :receiverID="null"/>
      </div>
    </div>
    <div v-else>
      <h2>Select user or group to chat</h2>
      <!-- Search input -->
      <input v-model="searchQuery" placeholder="Search users..."/>
      <div class="user-list">
        <h3>Users</h3>
        <div
            v-for="user in filteredUsers"
            :key="user.id"
            class="user-item"
            @click="startChatWithUser(user)"
        >
          <span :class="{ 'online-dot': user.status === 'online' }"></span>
          {{ user.username }}
          <!-- Add unread indicator -->
          <span v-if="getUnreadCount(user.id) > 0" class="unread-count">
    {{ formatUnreadCount(getUnreadCount(user.id)) }}
  </span>
        </div>
      </div>
      <!-- Group list -->
      <div class="group-list">
        <h3>Groups</h3>
        <div v-for="group in userGroups"
             :key="group.ID"
             class="group-item"
             @click="startChatWithGroup(group)">
          <span>{{ group.Name }}</span>
          <!-- Add debug info -->
          <small style="color: gray; margin-left: 5px;">(ID: {{ group.ID }})</small>
          <span v-if="getUnreadCount(group.ID) > 0"
                class="unread-count">
      {{ formatUnreadCount(getUnreadCount(group.ID)) }}
    </span>
        </div>
      </div>
    </div>

    <!-- Typing indicator (updated for groups)-->
    <div v-if="typingUsers.length > 0" class="typing-indicator">
      <span v-for="username in typingUsers" :key="username">{{
          username
        }}</span>
      is typing...
    </div>
  </div>
  <!-- Confirmation Modal -->
  <div v-if="showLeaveModal" class="modal-overlay" @click="showLeaveModal = false">
    <div class="modal-content" @click.stop>
      <h3>Leave Group</h3>
      <p>Are you sure you want to leave "{{ selectedGroup?.name }}"? This action cannot be undone.</p>
      <div class="modal-buttons">
        <button class="cancel" @click="showLeaveModal = false">Cancel</button>
        <button class="confirm" @click="leaveGroup">Leave Group</button>
      </div>
    </div>
  </div>

</template>

<script>
// frontend/src/components/ChatWindow.vue
import ChatMessages from "./ChatMessages.vue";
import ChatInput from "./ChatInput.vue";
import {useStore} from "vuex";
import axios from "axios";
import {computed, onBeforeUnmount, onMounted, ref, watch} from "vue";

export default {
  name: "ChatWindow",
  components: {
    ChatMessages,
    ChatInput,
  },
  setup() {
    const store = useStore();
    const instance = axios.create({
      baseURL: '/api', // Set base URL for all axios requests
    });
    // const ws = ref(null); // No longer needed as a ref here
    const currentUser = computed(() => store.getters.currentUser);
    const selectedUser = ref(null);
    const selectedGroup = ref(null);
    const usersOnline = computed(() => store.getters.getUsersOnline);
    const typingUsers = computed(() => store.getters.typingUsers);
    const searchQuery = ref("");
    const userGroups = ref([]);
    const copyMessage = ref("");
    const showLeaveModal = ref(false);
    //Get group member from store.
    const groupMembers = computed(() => store.getters.groupMembers);
    const loadingMembers = ref(false);
    const confirmLeaveGroup = () => {
      showLeaveModal.value = true;
    };
    const leaveGroup = async () => {
      try {
        if (!selectedGroup.value || !currentUser.value) return;

        await instance.post(`/groups/${selectedGroup.value.id}/leave`, {
          user_id: currentUser.value.id
        });

        // Send leave group message through WebSocket (check if ws exists)
        if (store.state.ws) {
          store.state.ws.send(JSON.stringify({
            type: "leave_group",
            group_id: selectedGroup.value.id
          }));
        }

        userGroups.value = userGroups.value.filter(g => g.ID.toString() !== selectedGroup.value.id);
        selectedGroup.value = null;
        store.dispatch('clearMessages');
        alert('You have left the group successfully');
      } catch (error) {
        console.error('Failed to leave group:', error);
        alert('Failed to leave group. Please try again.');
      } finally {
        showLeaveModal.value = false;
      }
    };

    const page= ref(1);
    const pageSize = ref(20);
    const hasMore = ref(true);
    const loadingMore = ref(false);
    const messagesContainer = ref(null);

    const handleScroll = () => {
      const container = messagesContainer.value;
      if (!container || loadingMore.value || !hasMore.value) return;

      if (container.scrollTop < 100) {
        loadMoreMessages();
      }
    };

    onMounted(async () => {
      if (currentUser.value) {
        connectWebSocket();
        await fetchAllUsers();
        await fetchUserGroups();
        store.dispatch('initializeUnreadCounts');
        debugUnreadCounts();
      }
    });

    // Clean up the WebSocket connection when the component is unmounted
    onBeforeUnmount(() => {
      if (store.state.ws) {
        store.state.ws.close();
        store.commit("setWs", null); // Reset to null
      }
    });

    watch(() => selectedGroup.value, (newGroup) => {
      if (newGroup) {
        store.dispatch('markAsRead', newGroup.id);
      }
    });

    const debugUnreadCounts = () => {
      userGroups.value.forEach(group => {
        store.getters.getUnreadCount(group.ID);
      });
    };

    const startChatWithUser = async (user) => {
      selectedUser.value = user;
      selectedGroup.value = null;
      store.dispatch("clearMessages");
      store.dispatch('markAsRead', user.id);
      page.value = 1;
      hasMore.value = true;
      await fetchMessages();
    };

    const startChatWithGroup = async (group) => {

      selectedGroup.value = {
        id: group.ID.toString(),
        name: group.Name,
        code: group.Code
      };

      selectedUser.value = null;
      store.dispatch('clearMessages');
      store.dispatch('markAsRead', group.ID.toString());

      // Join group via WebSocket (Check if ws exists)
      if (store.state.ws && group.ID) {
        store.state.ws.send(JSON.stringify({
          type: "join_group",
          group_id: group.ID
        }));
      }

      page.value = 1;
      hasMore.value = true;

      if (group.ID) {
        await fetchGroupMessages();
      }
    };

    const formatUnreadCount = (count) => {
      return count > 9 ? '9+' : count.toString();
    };

    const fetchAllUsers = async () => {
      instance.get(`/users`).then(res => {
        const users = res.data.map(user => ({
          id: user.id,
          username: user.username,
          status: 'offline',
        }));
        const filteredUsers = users.filter(user => user.id !== currentUser.value?.id);
        store.dispatch('setUsersOnline', filteredUsers)
      }).catch(() => {
      })
    }

    const fetchUserGroups = async () => {
      try {
        const response = await instance.get(
            `/users/${currentUser.value?.id}/groups`
        );
        userGroups.value = response.data;
      } catch (error) {
        console.error("Failed to fetch user's groups:", error);
      }
    };

    const connectWebSocket = () => {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const ws = new WebSocket(
          `${protocol}//${window.location.host}/api/ws?userID=${currentUser.value?.id}`
      );
      store.commit("setWs", ws); // Store the WebSocket instance in Vuex

      ws.onopen = () => {
        console.log("WebSocket connected");
        ws.send(JSON.stringify({type: "online_status"}));
        userGroups.value.forEach(group => {
          ws.send(JSON.stringify({
            type: "join_group",
            group_id: group.ID
          }));
        });
      };

      const updateMessageStatus = (messageId, status) => {
        if (ws) { // Check ws
          ws.send(JSON.stringify({
            type: 'message_status',
            message_id: messageId,
            status: status
          }));
        }
      };

      ws.onmessage = (event) => {
        try {
          const messages = event.data.split('\n').filter(msg => msg.trim());

          messages.forEach(message => {
            try {
              const data = JSON.parse(message);
              let messageObj = null;

              switch (data.type) {
                case "new_message":
                  // *** Include sender_username in messageObj ***
                  messageObj = {
                    id: data.message_id,
                    sender_id: data.sender_id,
                    sender_username: data.sender_username, // Add this line
                    receiver_id: data.receiver_id,
                    group_id: data.group_id,
                    content: data.content,
                    created_at: data.created_at,
                    reply_to_message_id: data.reply_to_message_id,
                    status: 'sent',
                    // Include file info
                    file_name: data.file_name,
                    file_path: data.file_path,
                    file_type: data.file_type,
                    file_size: data.file_size
                  };

                  if (data.reply_to_message) {
                    messageObj.reply_to_message = data.reply_to_message;
                  }
                  console.log("Received new_message via WebSocket:", messageObj); // ADD THIS

                  store.dispatch('addMessage', messageObj);

                  if (data.sender_id !== currentUser.value?.id) {
                    updateMessageStatus(data.message_id, 'delivered');
                    if (data.group_id) {
                      if (!selectedGroup.value || data.group_id.toString() !== selectedGroup.value.id.toString()) {
                        store.dispatch('incrementUnreadCount', data.group_id.toString());
                      }
                    } else {
                      if (!selectedUser.value || data.sender_id !== selectedUser.value.id) {
                        store.dispatch('incrementUnreadCount', data.sender_id.toString());
                      }
                    }
                  }
                  break;

                case "online_status":
                  if (data.user_id !== currentUser.value?.id) {
                    const userIndex = usersOnline.value.findIndex(u => u.id === data.user_id);
                    if (userIndex !== -1) {
                      const updatedUsers = [...usersOnline.value];
                      updatedUsers[userIndex].status = "online";
                      store.dispatch("setUsersOnline", updatedUsers);
                    } else {
                      instance.get(`/profile?userID=${data.user_id}`).then(res => {
                        const newUser = {id: res.data.id, username: res.data.username, status: 'online'};
                        store.dispatch('setUsersOnline', [...usersOnline.value, newUser]);
                      }).catch(err => console.error("Error fetching user profile", err));
                    }
                  }
                  break;

                case "offline_status":
                  if (data.user_id !== currentUser.value?.id) {
                    const userIndex = usersOnline.value.findIndex(u => u.id === data.user_id);
                    if (userIndex !== -1) {
                      const newUsers = [...usersOnline.value];
                      newUsers[userIndex].status = "offline"
                      store.dispatch("setUsersOnline", newUsers);
                    }
                  }
                  break;

                case "typing":
                  if (data.sender_id !== currentUser.value?.id) {
                    const sender = usersOnline.value.find(u => u.id === data.sender_id);
                    if (sender) {
                      store.dispatch('addTypingUser', sender.username);
                    }
                  }
                  break;

                case "stop_typing":
                  if (data.sender_id !== currentUser.value?.id) {
                    const sender = usersOnline.value.find(u => u.id === data.sender_id);
                    if (sender) {
                      store.dispatch("removeTypingUser", sender.username);
                    }
                  }
                  break;

                case "read_message":
                  break;

                case 'message_status':
                  store.dispatch('updateMessageStatus', {
                    messageId: data.message_id,
                    status: data.status
                  });
                  break;

                case "reaction_added":
                case "reaction_removed":
                  store.dispatch('updateReaction', {
                    messageId: data.message_id,
                    userId: data.user_id,
                    emoji: data.emoji,
                    type: data.type,
                  });
                  break;

                default:
                  console.log("Unhandled message type:", data.type);
                  break;
              }
            } catch (innerError) {
              console.error("Error parsing individual message:", innerError);
            }
          });
        } catch (error) {
          console.error("Error handling WebSocket message:", error);
        }
      };

      ws.onclose = () => {
        console.log("WebSocket disconnected");
        store.commit("setWs", null); // Set ws to null when closed
        setTimeout(connectWebSocket, 5000);
      };

      ws.onerror = (error) => {
        console.error("WebSocket error:", error);
      };
    };
    const fetchMessages = async () => {
      if (selectedUser.value) {
        try {
          const response = await instance.get(
              `/messages?user1=${currentUser.value?.id}&user2=${selectedUser.value?.id}&page=${page.value}&pageSize=${pageSize.value}`
          );
          store.commit('addMessages', response.data.messages.reverse());
          hasMore.value = (page.value * pageSize.value) < response.data.total;
        } catch (error) {
          console.error("Failed to fetch messages:", error);
        } finally {
          loadingMore.value = false;
        }
      }
    };
    const fetchGroupMessages = async () => {
      if (selectedGroup.value && selectedGroup.value.id) {
        try {
          const response = await instance.get(
              `/groups/${selectedGroup.value.id}/messages?page=${page.value}&pageSize=${pageSize.value}`
          );
          store.commit('addMessages', response.data.messages.reverse());
          hasMore.value = (page.value * pageSize.value) < response.data.total;
        } catch (error) {
          console.error("Failed to fetch group messages:", error);
        } finally {
          loadingMore.value = false;
        }
      }
    };

    const loadMoreMessages = async () => {
      if (!hasMore.value || loadingMore.value) return;

      loadingMore.value = true;

      try {
        page.value++;

        if (selectedUser.value) {
          await fetchMessages();
        } else if (selectedGroup.value) {
          await fetchGroupMessages();
        }

      } catch (error) {
        console.error('Error loading more messages:', error);
        page.value--;
      } finally {
        loadingMore.value = false;
      }
    };

    const filteredUsers = computed(() => {
      if (!searchQuery.value) {
        return usersOnline.value;
      }
      return usersOnline.value.filter((user) =>
          user.username.toLowerCase().includes(searchQuery.value.toLowerCase())
      );
    });

    const filteredMessages = computed(() => {
      if (selectedUser.value) {
        return store.getters.allMessages.filter(
            (message) =>
                (message.sender_id === currentUser.value?.id &&
                    message.receiver_id === selectedUser.value.id) ||
                (message.sender_id === selectedUser.value.id &&
                    message.receiver_id === currentUser.value?.id)
        );
      } else if (selectedGroup.value) {
        return store.getters.allMessages.filter(
            (message) => message.group_id === selectedGroup.value.id
        );
      }
      return [];
    });
    const copyCode = (code) => {
      navigator.clipboard.writeText(code)
          .then(() => {
            copyMessage.value = 'Code copied to clipboard!';
            setTimeout(() => {
              copyMessage.value = '';
            }, 2000);
          })
          .catch(() => {
            copyMessage.value = 'Failed to copy code';
            setTimeout(() => {
              copyMessage.value = '';
            }, 2000);
          });
    };

    //Add isUserOnline function.
    const isUserOnline = (userId) => {
      return store.getters.getUsersOnline.some(user => user.id === userId && user.status === 'online');
    };
    // Watch for changes in the selectedGroup
    watch(
        () => selectedGroup.value,
        async (newGroup, oldGroup) => {
          //If change to another group. Clean old members.
          if(oldGroup && oldGroup.id){
            store.dispatch('clearGroupMembers');
          }

          if (newGroup && newGroup.id) {
            loadingMembers.value = true; // Set loading to true
            await store.dispatch("fetchGroupMembers", newGroup.id);
            loadingMembers.value = false;// Set loading to false
          } else {
            store.dispatch("clearGroupMembers");
          }
        }, {deep: true}
    );


    return {
      currentUser,
      // ws, // No need to return this directly
      connectWebSocket,
      selectedUser,
      selectedGroup,
      usersOnline,
      startChatWithUser,
      startChatWithGroup,
      typingUsers,
      searchQuery,
      filteredUsers,
      userGroups,
      filteredMessages,
      copyCode,
      copyMessage,
      hasMore,
      loadMoreMessages,
      messagesContainer,
      handleScroll,
      loadingMore,
      formatUnreadCount,
      fetchGroupMessages,
      fetchMessages,
      isUserOnline,
      getUnreadCount: (id) => store.getters.getUnreadCount(id),
      showLeaveModal,
      confirmLeaveGroup,
      leaveGroup,
      groupMembers,
      loadingMembers,
    };
  },
};
</script>

<style scoped>
.chat-container {
  display: flex;
  flex-direction: column;
  height: 400px;
}

.chat-messages {
  height: 400px;
  overflow-y: auto;
  padding: 1rem;
  border: 1px solid #ccc;
  border-radius: 4px;
  margin: 1rem 0;
}

.loading-indicator {
  text-align: center;
  padding: 1rem;
  color: #666;
  background-color: #f5f5f5;
  border-radius: 4px;
  margin-bottom: 0.5rem;
}

.user-list {
  display: flex;
  flex-wrap: wrap;
  margin-top: 20px; /* Add some space above the user list */
}

.user-item {
  padding: 10px;
  margin: 5px;
  border: 1px solid #ccc;
  border-radius: 5px;
  cursor: pointer;
  display: flex;
  align-items: center;
  position: relative; /* Needed for absolute positioning of .unread-count */
}

.group-list {
  margin-top: 20px;
  border-top: 1px solid #ccc;
  padding-top: 10px;
}

.group-item {
  padding: 10px;
  margin: 5px;
  border: 1px solid #ccc;
  border-radius: 5px;
  cursor: pointer;
  display: flex;
  align-items: center;
  position: relative; /* Needed for absolute positioning of .unread-count */
}

.online-dot {
  width: 10px;
  height: 10px;
  background-color: green;
  border-radius: 50%;
  margin-right: 5px;
  display: inline-block;
}

.typing-indicator {
  font-style: italic;
  color: gray;
  margin-top: 5px;
}

/* Add styles for profile info */
.profile-info {
  padding: 10px;
  border-bottom: 1px solid #ccc;
  margin-bottom: 10px;
}

/* Style for the unread count indicator */
.unread-count {
  background-color: red;
  color: white;
  border-radius: 12px; /* Increased to look better with 9+ */
  padding: 2px;
  font-size: 12px;
  position: absolute;
  top: 5px;
  right: 5px;
  min-width: 18px; /* Ensure consistent width */
  text-align: center;
}

.copy-button {
  padding: 4px 12px;
  background-color: #007bff;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.8em;
}

.copy-button:hover {
  background-color: #0056b3;
}

.copy-message {
  font-size: 0.8em;
  color: #28a745;
  font-weight: normal;
}

.group-header {
  padding: 15px;
  border-bottom: 1px solid #eee;
  display: flex;
  align-items: center;
  background-color: #f8f9fa;
}

.group-header h2 {
  margin: 0;
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.group-code {
  font-size: 0.9em;
  color: #666;
  background-color: #e9ecef;
  padding: 4px 8px;
  border-radius: 4px;
  font-weight: normal;
  position: relative;
}

.group-code-container {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.group-code:hover::before {
  content: "Share this code to invite others";
  position: absolute;
  bottom: 100%;
  left: 50%;
  transform: translateX(-50%);
  padding: 5px 10px;
  background-color: #333;
  color: white;
  border-radius: 4px;
  font-size: 0.8em;
  white-space: nowrap;
  pointer-events: none;
  opacity: 1;
  transition: opacity 0.2s;
}

.leave-button {
  padding: 4px 12px;
  background-color: #dc3545;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.8em;
  transition: background-color 0.2s;
}

.leave-button:hover {
  background-color: #c82333;
}

.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.5);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
}

.modal-content {
  background-color: white;
  padding: 20px;
  border-radius: 8px;
  max-width: 400px;
  width: 90%;
}

.modal-buttons {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 20px;
}

.modal-buttons button {
  padding: 8px 16px;
  border-radius: 4px;
  border: none;
  cursor: pointer;
}

.modal-buttons .cancel {
  background-color: #6c757d;
  color: white;
}

.modal-buttons .confirm {
  background-color: #dc3545;
  color: white;
}
/* Add style  */
.group-members {
  margin-top: 10px;
  padding: 10px;
  border-top: 1px solid #eee;
}

.group-members h3 {
  margin-bottom: 5px;
}

.group-members ul {
  list-style: none;
  padding: 0;
}

.group-members li {
  padding: 5px 0;
  display: flex;
  align-items: center;
}
/* Add loading  */
.loading-members {
  font-style: italic;
  color: gray;
}
</style>
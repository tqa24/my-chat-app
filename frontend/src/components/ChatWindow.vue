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
        <ChatMessages :messages="filteredMessages"/>
        <ChatInput :receiverID="selectedUser.id" :groupID="null"/>
      </div>
      <!-- Chatting with a group -->
      <div v-else-if="selectedGroup">
        <div class="group-header">
          <h2>
            {{ selectedGroup.name }}
            <div class="group-code-container">
      <span class="group-code" title="Share this code to invite others">
        Group Code: {{ selectedGroup.code }}
      </span>
              <button class="copy-button" @click.stop="copyCode(selectedGroup.code)"
                      title="Copy to clipboard">
                Copy Code
              </button>
            </div>
            <span v-if="copyMessage" class="copy-message">{{ copyMessage }}</span>
          </h2>
        </div>
        <ChatMessages :messages="filteredMessages"/>
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
</template>

<script>
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
    const ws = ref(null);
    const currentUser = computed(() => store.getters.currentUser);
    const selectedUser = ref(null); //Selected user to chat
    const selectedGroup = ref(null); // Selected group to chat
    const usersOnline = computed(() => store.getters.getUsersOnline);
    const typingUsers = computed(() => store.getters.typingUsers);
    const searchQuery = ref(""); // Add search query
    const userGroups = ref([]); // To store user's groups
    const copyMessage = ref("");

    onMounted(async () => {
      if (currentUser.value) {
        connectWebSocket();
        await fetchAllUsers(); // Fetch all users when component mounts
        await fetchUserGroups(); // Fetch user's groups
        store.dispatch('initializeUnreadCounts'); // Initialize unread counts
        debugUnreadCounts(); // Debug unread counts
      }
    });

    onBeforeUnmount(() => {
      if (ws.value) {
        ws.value.close();
      }
    });

    watch(() => selectedGroup.value, (newGroup, oldGroup) => {
      console.log('Selected group changed:', {
        from: oldGroup?.id,
        to: newGroup?.id
      });

      if (newGroup) {
        console.log('Current unread count for new group:',
            store.getters.getUnreadCount(newGroup.id));
        store.dispatch('markAsRead', newGroup.id);
      }
    });

    const debugUnreadCounts = () => {
      const counts = store.state.unreadCounts;
      console.log('Current unread counts:', counts);
      userGroups.value.forEach(group => {
        console.log(`Group ${group.Name} (${group.ID}):`,
            store.getters.getUnreadCount(group.ID));
      });
    };

    // --- User selection ---
    const startChatWithUser = async (user) => {
      selectedUser.value = user;
      selectedGroup.value = null; // Clear selected group
      store.dispatch("clearMessages"); // Clear previous messages
      store.dispatch('markAsRead', user.id); // Mark messages as read
      await fetchMessages(); // Fetch messages for the new conversation
    };

    // --- Group Selection ---
    const startChatWithGroup = async (group) => {
      console.log('Starting group chat with:', {
        group,
        groupId: group.ID,
        currentUnreadCount: store.getters.getUnreadCount(group.ID)
      });

      selectedGroup.value = {
        id: group.ID.toString(),
        name: group.Name,
        code: group.Code
      };

      selectedUser.value = null;
      store.dispatch('clearMessages');

      // Mark messages as read
      store.dispatch('markAsRead', group.ID.toString());

      // Join group via WebSocket
      if (ws.value && group.ID) {
        ws.value.send(JSON.stringify({
          type: "join_group",
          group_id: group.ID
        }));
      }

      if (group.ID) {
        await fetchGroupMessages();
      }
    };


    // Add format function for unread counts
    const formatUnreadCount = (count) => {
      return count > 9 ? '9+' : count.toString();
    };

    // --- Fetch Users ---
    const fetchAllUsers = async () => {
      //Get all user and add to user online
      axios.get(`http://localhost:8080/users`).then(res => {
        const users = res.data.map(user => ({
          id: user.id,
          username: user.username,
          status: 'offline', // Assume offline initially
        }));
        // Remove current user
        const filteredUsers = users.filter(user => user.id !== currentUser.value?.id);
        store.dispatch('setUsersOnline', filteredUsers)
      }).catch(err => {
        console.error("Error", err)
      })
    }

    // --- Fetch User's Groups ---
    const fetchUserGroups = async () => {
      try {
        const response = await axios.get(
            `http://localhost:8080/users/${currentUser.value?.id}/groups`
        );
        userGroups.value = response.data;
      } catch (error) {
        console.error("Failed to fetch user's groups:", error);
      }
    };

    const connectWebSocket = () => {
      ws.value = new WebSocket(
          `ws://localhost:8080/ws?userID=${currentUser.value?.id}`
      );
      store.commit("setWs", ws.value);
      ws.value.onopen = () => {
        console.log("WebSocket connected");
        // Send "online_status" event
        ws.value.send(JSON.stringify({type: "online_status"}));
        // Join all groups
        userGroups.value.forEach(group => {
          ws.value.send(JSON.stringify({
            type: "join_group",
            group_id: group.ID
          }));
        });
      };

      ws.value.onmessage = (event) => {
        try {
          // Split the message if it contains multiple JSON objects
          const messages = event.data.split('\n').filter(msg => msg.trim());

          messages.forEach(message => {
            try {
              const data = JSON.parse(message);
              console.log("Received:", data);

              switch (data.type) {
                case "new_message":
                  store.dispatch('addMessage', data);
                  if (data.sender_id !== currentUser.value?.id) {
                    // For group messages
                    if (data.group_id) {
                      console.log('Group message received:', {
                        message: data,
                        group_id: data.group_id,
                        selected_group: selectedGroup.value?.id,
                        current_unread: store.getters.getUnreadCount(data.group_id),
                        is_current_group: selectedGroup.value?.id === data.group_id.toString()
                      });

                      // Only increment if we're not currently viewing this group
                      if (!selectedGroup.value || data.group_id.toString() !== selectedGroup.value.id.toString()) {
                        console.log('Incrementing unread count for group:', data.group_id);
                        store.dispatch('incrementUnreadCount', data.group_id.toString());
                      }
                    }
                    // For direct messages
                    else {
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
                      axios.get(`http://localhost:8080/profile?userID=${data.user_id}`).then(res => {
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

                default:
                  console.log("Unhandled message type:", data.type);
                  break;
              }
            } catch (innerError) {
              console.error("Error parsing individual message:", innerError);
              console.log("Problematic message:", message);
            }
          });
        } catch (error) {
          console.error("Error handling WebSocket message:", error);
          console.log("Raw message:", event.data);
        }
      };


      ws.value.onclose = () => {
        console.log("WebSocket disconnected");
        store.commit("setWs", null);
        // Attempt to reconnect (optional)
        setTimeout(connectWebSocket, 5000); // Retry after 5 seconds
      };

      ws.value.onerror = (error) => {
        console.error("WebSocket error:", error);
      };
    };
    const fetchMessages = async () => {
      // Only fetch messages if a user is selected
      if (selectedUser.value) {
        try {
          const response = await axios.get(
              `http://localhost:8080/messages?user1=${currentUser.value?.id}&user2=${selectedUser.value?.id}`
          );
          store.dispatch('setMessages', response.data);
        } catch (error) {
          console.error("Failed to fetch messages:", error);
        }
      }
    };

    const fetchGroupMessages = async () => {
      if (selectedGroup.value && selectedGroup.value.id) {
        try {
          console.log("Fetching messages for group:", selectedGroup.value.id);
          const response = await axios.get(
              `http://localhost:8080/groups/${selectedGroup.value.id}/messages`
          );
          store.dispatch('setMessages', response.data);
        } catch (error) {
          console.error("Failed to fetch group messages:", error);
        }
      } else {
        console.error("No group selected or invalid group ID");
      }
    };

    // Computed property for filtered users
    const filteredUsers = computed(() => {
      if (!searchQuery.value) {
        return usersOnline.value; // Return all users if no search query
      }
      return usersOnline.value.filter((user) =>
          user.username.toLowerCase().includes(searchQuery.value.toLowerCase())
      );
    });
    // Computed property for filtered messages based on selected user/group
    const filteredMessages = computed(() => {
      if (selectedUser.value) {
        // Filter messages for direct chat with the selected user
        return store.getters.allMessages.filter(
            (message) =>
                (message.sender_id === currentUser.value?.id &&
                    message.receiver_id === selectedUser.value.id) ||
                (message.sender_id === selectedUser.value.id &&
                    message.receiver_id === currentUser.value?.id)
        );
      } else if (selectedGroup.value) {
        // Filter messages for the selected group
        return store.getters.allMessages.filter(
            (message) => message.group_id === selectedGroup.value.id
        );
      }
      return []; // Return an empty array if no user or group is selected
    });
    const copyCode = (code) => {
      navigator.clipboard.writeText(code)
          .then(() => {
            copyMessage.value = 'Code copied to clipboard!';
            setTimeout(() => {
              copyMessage.value = '';
            }, 2000);
          })
          .catch(err => {
            console.error('Failed to copy code: ', err);
            copyMessage.value = 'Failed to copy code';
            setTimeout(() => {
              copyMessage.value = '';
            }, 2000);
          });
    };

    return {
      currentUser,
      ws,
      connectWebSocket,
      selectedUser,
      selectedGroup,
      usersOnline,
      startChatWithUser,
      startChatWithGroup,
      typingUsers,
      fetchMessages,
      fetchGroupMessages,
      searchQuery,
      filteredUsers,
      userGroups,
      filteredMessages,
      copyCode,
      copyMessage,
      formatUnreadCount,
      getUnreadCount: (id) => store.getters.getUnreadCount(id),
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
  flex-grow: 1;
  overflow-y: auto;
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
  transition: background-color 0.2s;
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
  opacity: 0;
  transition: opacity 0.2s;
}

.group-code:hover::before {
  opacity: 1;
}
</style>
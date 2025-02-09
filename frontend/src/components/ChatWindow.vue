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
        <ChatMessages />
        <ChatInput :receiverID="selectedUser.id" :groupID="null"/>
      </div>
      <!-- Chatting with a group -->
      <div v-else-if="selectedGroup">
        <h2>Chatting with {{ selectedGroup.name }}</h2>
        <ChatMessages />
        <ChatInput :groupID="selectedGroup.id" :receiverID="null"/>
      </div>
    </div>
    <div v-else>
      <h2>Select user or group to chat</h2>
      <!-- Search input -->
      <input v-model="searchQuery" placeholder="Search users..." />
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
        </div>
      </div>
      <!-- Group list -->
      <div class="group-list">
        <h3>Groups</h3>
        <div v-for="group in userGroups" :key="group.id" class="group-item" @click="startChatWithGroup(group)">
          <span>{{ group.name }}</span>
        </div>
      </div>
    </div>

    <!-- Typing indicator (updated for groups)-->
    <div v-if="typingUsers.length > 0" class="typing-indicator">
        <span v-for="username in typingUsers" :key="username">
            {{ username }} is typing...
        </span>
    </div>
  </div>
</template>

<script>
import ChatMessages from "./ChatMessages.vue";
import ChatInput from "./ChatInput.vue";
import { computed, ref, onMounted, onBeforeUnmount } from "vue";
import { useStore } from "vuex";
import axios from "axios";

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
    const selectedUser = ref(null); // Selected user to chat
    const selectedGroup = ref(null); // Selected group to chat
    const usersOnline = computed(() => store.getters.getUsersOnline);
    const typingUsers = computed(() => store.getters.typingUsers);
    const searchQuery = ref(""); // Add search query
    const userGroups = ref([]); // To store user's groups

    onMounted(async () => {
      if (currentUser.value) { // Check if user is logged in
        connectWebSocket();
        fetchAllUsers(); // Fetch all users
        fetchUserGroups(); // Fetch user's groups
      }
    });

    onBeforeUnmount(() => {
      if (ws.value) {
        ws.value.close();
      }
    });

    // --- User selection ---
    const startChatWithUser = async (user) => {
      selectedUser.value = user;
      selectedGroup.value = null; // Clear selected group
      store.dispatch("clearMessages"); // Clear previous messages
      await fetchMessages(); // Fetch messages for the new conversation
    };

    // --- Group Selection ---
    const startChatWithGroup = async (group) => {
      selectedGroup.value = group;
      selectedUser.value = null; // Clear selected user
      store.dispatch('clearMessages'); // Clear existing messages
      await fetchGroupMessages();  // Fetch messages for the selected group
    };

    // --- Fetch Users ---
    const fetchAllUsers = async () => {
      //Get all user and add to user online
      axios.get(`http://localhost:8080/users`).then(res => { //Removed await

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
    };

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
      if (!currentUser.value) {
        return; // Don't connect if not logged in
      }
      ws.value = new WebSocket(
          `ws://localhost:8080/ws?userID=${currentUser.value.id}`
      );
      store.commit("setWs", ws.value);
      ws.value.onopen = () => {
        console.log("WebSocket connected");
        // Send "online_status" event
        ws.value.send(JSON.stringify({ type: "online_status" }));
      };

      ws.value.onmessage = (event) => {
        const data = JSON.parse(event.data);
        console.log("Received:", data);
        // Handle different message types
        switch (data.type) {
          case "new_message": {
            // Add new message to the store
            // Only show if have select user and match sender and receive
            if(selectedUser.value && ((data.sender_id === selectedUser.value.id && data.receiver_id === currentUser.value.id) || (data.sender_id === currentUser.value.id && data.receiver_id === selectedUser.value.id) )){
              store.dispatch("addMessage", data);
            }
            //Show message if it belong to selected group
            if(selectedGroup.value && data.group_id === selectedGroup.value.id){
              store.dispatch("addMessage", data)
            }
            break;
          }
          case "online_status": {
            // Update online users list
            if (data.user_id !== currentUser.value?.id) {//Add ?
              //Find in user list
              const userIndex = usersOnline.value.findIndex(u => u.id === data.user_id)
              if(userIndex !== -1){
                const updatedUsers = [...usersOnline.value];
                updatedUsers[userIndex].status = "online"
                store.dispatch("setUsersOnline", updatedUsers);
              }
            }

            break;
          }
          case "offline_status": {
            // Update users online
            const currentUsers = store.getters.getUsersOnline;
            const index = currentUsers.findIndex((u) => u.id === data.user_id);
            if (index !== -1) {
              const newUsers = [...currentUsers];
              newUsers[index].status = "offline"
              store.dispatch("setUsersOnline", newUsers);
            }
            break;
          }
          case "typing": {
            // Add typing user to the store (if it's not the current user)
            if (data.sender_id !== currentUser.value?.id) {//Add ?
              //Find sender from user online
              const sender = usersOnline.value.find(u => u.id === data.sender_id)
              if(sender){
                store.dispatch("addTypingUser", sender.username); // Or use sender's username
              }
            }
            break;
          }
          case "stop_typing": {
            // Remove typing user from the store and use username
            if (data.sender_id !== currentUser.value?.id) {
              //Find sender from user online
              const sender = usersOnline.value.find(u => u.id === data.sender_id)
              if(sender){
                store.dispatch("removeTypingUser", sender.username);
              }
            }
            break;
          }
          case "read_message": {
            // Handle message read status (update your message objects)
            break;
          }
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
      try {
        const response = await axios.get(
            `http://localhost:8080/messages?user1=${currentUser.value?.id}&user2=${selectedUser.value?.id}` // Add ? here
        );
        store.dispatch("setMessages", response.data);
      } catch (error) {
        console.error("Failed to fetch messages:", error);
      }
    };

    const fetchGroupMessages = async () => {
      try {
        const response = await axios.get(
            `http://localhost:8080/groups/${selectedGroup.value.id}/messages`
        );
        store.dispatch("setMessages", response.data);
      } catch (error) {
        console.error("Failed to fetch group messages:", error);
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
      searchQuery, // Expose search query
      filteredUsers, // Expose filtered users
      userGroups
    };
  },
};
</script>

<style scoped>
.chat-container {
  display: flex;
  flex-direction: column;
  height: 400px; /* Or whatever height you want */
}

.chat-messages {
  flex-grow: 1; /* Allow messages to take up available space */
  overflow-y: auto; /* Add scrolling */
}
.user-list {
  display: flex;
  flex-wrap: wrap;
}

.user-item {
  padding: 10px;
  margin: 5px;
  border: 1px solid #ccc;
  border-radius: 5px;
  cursor: pointer;
  display: flex;
  align-items: center;
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
/* Style for group list */
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
}
</style>
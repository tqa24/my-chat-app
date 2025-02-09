<template>
  <div class="chat-container">
    <div v-if="selectedUser">
      <h2>Chatting with {{ selectedUser.username }}</h2>
      <ChatMessages />
      <ChatInput :receiverID="selectedUser.id"/>
    </div>
    <div v-else>
      <h2>Select user to chat</h2>
      <div class="user-list">
        <div v-for="user in usersOnline" :key="user.id" class="user-item" @click="startChat(user)">
          <span :class="{ 'online-dot': user.status === 'online' }"></span>
          {{ user.username }}
        </div>
      </div>
    </div>

    <!-- Typing indicator -->
    <div v-if="typingUsers.length > 0" class="typing-indicator">
      <span v-for="username in typingUsers" :key="username">{{ username }} is typing...</span>
    </div>
  </div>
</template>
<script>
import ChatMessages from './ChatMessages.vue';
import ChatInput from './ChatInput.vue';
import { computed, ref, onMounted, onBeforeUnmount } from 'vue';
import { useStore } from 'vuex';
import axios from 'axios';

export default {
  components: {
    ChatMessages,
    ChatInput,
  },
  setup() {
    const store = useStore();
    const ws = ref(null);
    const currentUser = computed(() => store.getters.currentUser);
    const selectedUser = ref(null) //Selected user to chat
    const usersOnline = computed(() => store.getters.getUsersOnline)
    const typingUsers = computed(() => store.getters.typingUsers)

    onMounted(async () => {
      if(currentUser.value){
        connectWebSocket();
// Fetch initial messages (if you have a selected user)
        if (selectedUser.value) {
          await fetchMessages();
        }
      }

    });

    onBeforeUnmount(() => {
      if (ws.value) {
        ws.value.close();
      }
    });
    const startChat = async (user) => {
      selectedUser.value = user;
      store.dispatch('clearMessages')// Clear previous messages
      await fetchMessages(); // Fetch messages for the new conversation

    };

    const connectWebSocket = () => {
      ws.value = new WebSocket(`ws://localhost:8080/ws?userID=${currentUser.value.id}`);
      store.commit('setWs', ws.value);
      ws.value.onopen = () => {
        console.log('WebSocket connected');
// Send "online_status" event
        ws.value.send(JSON.stringify({ type: "online_status" }));
      };

      ws.value.onmessage = (event) => {
        const data = JSON.parse(event.data);
        console.log("Received:", data);
        // Handle different message types
        switch (data.type) {
          case "new_message": { // Add braces here
            // Add new message to the store
            store.dispatch('addMessage', data);
            break;
          }
          case 'online_status': { // And here
            // Update online users list
            if(data.user_id !== currentUser.value.id){
              axios.get(`http://localhost:8080/profile?userID=${data.user_id}`).then((res)=>{
                const newUser = { id: data.user_id, username: res.data.username, status: 'online' };
                const currentUsers = store.getters.getUsersOnline
                const existingUserIndex = currentUsers.findIndex(u => u.id === data.user_id);
                if (existingUserIndex !== -1) {
                  // Update existing user
                  const updatedUsers = [...currentUsers];
                  updatedUsers[existingUserIndex] = newUser;
                  store.dispatch('setUsersOnline', updatedUsers);
                } else {
                  // Add new user
                  store.dispatch('setUsersOnline', [...currentUsers, newUser]);
                }
              }).catch(err => {
                console.log(err)
              })
            }

            break;
          }
          case 'offline_status': { // And here
            // Update users online
            const currentUsers = store.getters.getUsersOnline
            const index = currentUsers.findIndex(u=> u.id === data.user_id)
            if(index !== -1){
              const newUsers =[...currentUsers]
              newUsers.splice(index, 1)
              store.dispatch('setUsersOnline', newUsers)
            }
            break;
          }
          case "typing": { // And here
            // Add typing user to the store (if it's not the current user)
            if (data.sender_id !== currentUser.value.id) {
              store.dispatch('addTypingUser', data.sender_id); // Or use sender's username
            }
            break;
          }
          case "stop_typing": { // And here
            // Remove typing user from the store
            store.dispatch('removeTypingUser', data.sender_id);
            break;
          }
          case "read_message": { // And here
            // Handle message read status (update your message objects)
            break;
          }
        }
      };

      ws.value.onclose = () => {
        console.log('WebSocket disconnected');
        store.commit('setWs', null)
// Attempt to reconnect (optional)
        setTimeout(connectWebSocket, 5000); // Retry after 5 seconds
      };

      ws.value.onerror = (error) => {
        console.error('WebSocket error:', error);
      };
    };
    const fetchMessages = async () => {
      try {
        const response = await axios.get(`http://localhost:8080/messages?user1=${currentUser.value.id}&user2=${selectedUser.value.id}`);
        store.dispatch('setMessages', response.data);
      } catch (error) {
        console.error('Failed to fetch messages:', error);
      }
    };

    return { currentUser, ws, connectWebSocket, selectedUser, usersOnline, startChat, typingUsers, fetchMessages };
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
  align-items: center
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
</style>
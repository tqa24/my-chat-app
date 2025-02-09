<template>
  <div class="chat-messages">
    <div v-for="message in messages" :key="message.id" :class="messageClass(message)">
      <div class="message-content">
        <span>{{ message.content }}</span>
      </div>
      <div class="message-meta">
        {{ formatTime(message.created_at) }} - {{ message.status }} - {{ message.sender_id == currentUser?.id? currentUser.username: userIdToName[message.sender_id]}}
      </div>
    </div>
  </div>
</template>

<script>
import { computed } from 'vue';
import { useStore } from 'vuex';
import { format } from 'date-fns';

export default {
  setup() {
    const store = useStore();
    const messages = computed(() => store.getters.allMessages);
    const currentUser = computed(() => store.getters.currentUser);
    const usersOnline = computed(()=> store.getters.getUsersOnline)
    const userIdToName = computed(()=>{
      const map = {};
      usersOnline.value.forEach(user => {
        map[user.id] = user.username;
      });
      return map
    })

    const messageClass = (message) => {
      return {
        'message': true,
        // Use optional chaining (?.) to safely access currentUser.value.id
        'sent-message': currentUser.value?.id === message.sender_id,
        'received-message': currentUser.value?.id !== message.sender_id,
      };
    };

    const formatTime = (timestamp) => {
      return format(new Date(timestamp), 'HH:mm');
    };

    return { messages, currentUser, messageClass, formatTime, userIdToName };
  }
};
</script>

<style scoped>
.chat-messages {
  padding: 10px;
  overflow-y: auto; /* Enable scrolling */
  height: 300px; /* Set a fixed height */
  border: 1px solid #ccc;
}
.message {
  margin-bottom: 10px;
  padding: 8px;
  border-radius: 5px;
}
.sent-message {
  background-color: #dcf8c6; /* Light green for sent messages */
  align-self: flex-end; /* Align to the right */
}

.received-message {
  background-color: #f0f0f0; /* Light gray for received messages */
  align-self: flex-start; /* Align to the left */
}
.message-content {
  margin-bottom: 4px;
}
.message-meta{
  font-size: 12px;
  color: gray
}
</style>
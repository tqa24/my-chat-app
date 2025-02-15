<template>
  <div class="chat-messages">
    <div v-for="message in reversedMessages" :key="message.id" :class="messageClass(message)">
      <div class="message-content">
        <span>{{ message.content }}</span>
      </div>
      <div class="message-meta">
        {{ formatTime(message.created_at) }} - {{ message.status }} -
        {{ message.sender_id == currentUser?.id ? currentUser.username : userIdToName[message.sender_id] }}
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
    const usersOnline = computed(()=> store.getters.getUsersOnline);
    const reversedMessages = computed(() => [...messages.value].reverse());
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

    return { reversedMessages, messages, currentUser, messageClass, formatTime, userIdToName };
  }
};
</script>

<style scoped>
.chat-messages {
  padding: 10px;
  overflow-y: auto;
  height: 300px;
  display: flex;
  flex-direction: column-reverse; /* This makes newest messages appear at bottom */
}

.message {
  margin: 4px 0;
  padding: 8px 12px;
  border-radius: 18px;
  max-width: 70%;
  word-wrap: break-word;
}

.sent-message {
  background-color: #0084ff;
  color: white;
  align-self: flex-end;
  margin-left: auto;
}

.received-message {
  background-color: #f0f0f0;
  color: black;
  align-self: flex-start;
  margin-right: auto;
}

.message-meta {
  font-size: 11px;
  opacity: 0.7;
  margin-top: 4px;
}
</style>
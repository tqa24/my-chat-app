<template>
  <div class="messages" ref="messagesContainer">
    <div v-for="message in messages"
         :key="message.id"
         :class="['message', getMessageClass(message)]"
         :data-message-id="message.id">
      <!-- Add sender's username for group messages -->
      <div class="message-sender" v-if="message.group_id && message.sender_id !== currentUser?.id">
        {{ getSenderUsername(message) }}
      </div>
      <div class="message-content">

        <!-- Reply preview section -->
        <div v-if="message.reply_to_message" class="reply-preview" @click="scrollToMessage(message.reply_to_message.id)">
          <span>{{ getReplyPreview(message.reply_to_message) }}</span>
        </div>

        <div class="message-text">{{ message.content }}</div>

        <!-- Reactions section -->
        <div class="reactions-container">
          <!-- Display existing reactions -->
          <div v-if="message.reactions && Object.keys(message.reactions).length > 0" class="reactions">
            <span v-for="(users, emoji) in message.reactions"
                  :key="emoji"
                  class="reaction"
                  :class="{ 'user-reacted': hasUserReactedWithEmoji(message, emoji) }"
                  :title="getReactionUsers(users)"
                  @click="handleReactionClick(message, emoji)">
              {{ emoji }} {{ users.length }}
            </span>
          </div>
        </div>

        <!-- Message actions -->
        <div class="message-actions">
          <div class="action-buttons">
            <button class="action-button" @click="replyToMessage(message)" title="Reply">
              ↩️
            </button>
            <button class="action-button" @click.stop="toggleReactionPicker(message)" title="React">
              😀
            </button>
          </div>

          <!-- Reaction picker -->
          <div v-if="showReactionPicker && selectedMessageId === message.id"
               class="reaction-picker"
               v-click-outside="closeReactionPicker">
            <span v-for="emoji in availableReactions"
                  :key="emoji"
                  @click="addReaction(message, emoji)"
                  :class="{ 'selected': hasUserReactedWithEmoji(message, emoji) }"
                  class="emoji-option">
                {{ emoji }}
            </span>
          </div>
        </div>
      </div>
      <div class="message-info">
        <small>{{ formatTime(message.created_at) }}</small>
        <span class="message-status" :class="message.status">
        {{ getStatusIcon(message.status) }}
      </span>
      </div>
    </div>
  </div>
</template>

<script>
import { ref, computed, onMounted, nextTick, watch } from 'vue';
import { useStore } from 'vuex';

export default {
  props: {
    messages: {
      type: Array,
      required: true
    }
  },
  directives: {
    'click-outside': {
      mounted(el, binding) {
        el._clickOutside = (event) => {
          console.log('Click detected:', event); // ADD THIS LINE
          if (!(el === event.target || el.contains(event.target))) {
            binding.value(event);
          }
        };
        document.addEventListener('click', el._clickOutside);
      },
      unmounted(el) {
        document.removeEventListener('click', el._clickOutside);
      },
    },
  },
  setup(props) {
    const store = useStore();
    const showReactionPicker = ref(false);
    const selectedMessageId = ref(null);
    const hoveredMessageId = ref(null);
    const messagesContainer = ref(null);
    const currentUser = computed(() => store.state.user);

    const availableReactions = ['👍', '❤️','😂', '😮', '😢', '😠'];

    // Scroll to bottom when new messages arrive
    watch(() => props.messages, async () => {
      await nextTick();
      scrollToBottom();
    }, { deep: true });

    onMounted(() => {
      scrollToBottom();
    });

    const scrollToBottom = () => {
      if (messagesContainer.value) {
        messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight;
      }
    };

    const closeReactionPicker = () => {
      console.log('closeReactionPicker called'); // ADD THIS
      showReactionPicker.value = false;
      selectedMessageId.value = null;
    };

    const toggleReactionPicker = (message) => {
      console.log('toggleReactionPicker called with message:', message); // ADD THIS
      console.log('Current selectedMessageId:', selectedMessageId.value); // ADD THIS

      if (selectedMessageId.value === message.id) {
        console.log('Closing picker (selectedMessageId matches)'); // ADD THIS
        closeReactionPicker();
      } else {
        console.log('Opening picker (selectedMessageId does NOT match)'); // ADD THIS
        showReactionPicker.value = true;
        selectedMessageId.value = message.id;
        console.log('New selectedMessageId:', selectedMessageId.value); // ADD THIS
      }
    };

    const handleReactionClick = (message, emoji) => {
      if (hasUserReactedWithEmoji(message, emoji)) {
        removeReaction(message, emoji);
      } else {
        addReaction(message, emoji);
      }
    };

    const addReaction = async (message, emoji) => {
      try {
        // Remove existing reaction if any
        Object.keys(message.reactions || {}).forEach(existingEmoji => {
          if (message.reactions[existingEmoji].includes(currentUser.value.id)) {
            removeReaction(message, existingEmoji);
          }
        });

        // Send reaction to backend via WebSocket
        store.state.ws.send(JSON.stringify({
          type: "reaction",
          message_id: message.id,
          emoji: emoji
        }));

        // Optimistically update UI
        if (!message.reactions) {
          message.reactions = {};
        }
        if (!message.reactions[emoji]) {
          message.reactions[emoji] = [];
        }
        if (!message.reactions[emoji].includes(currentUser.value.id)) {
          message.reactions[emoji].push(currentUser.value.id);
        }
      } catch (error) {
        console.error('Failed to add reaction:', error);
      }
      closeReactionPicker();
    };

    const removeReaction = (message, emoji) => {
      try {
        // Send remove reaction to backend via WebSocket
        store.state.ws.send(JSON.stringify({
          type: "remove_reaction",
          message_id: message.id,
          emoji: emoji
        }));

        // Optimistically update UI
        if (message.reactions?.[emoji]) {
          const index = message.reactions[emoji].indexOf(currentUser.value.id);
          if (index > -1) {
            message.reactions[emoji].splice(index, 1);
            if (message.reactions[emoji].length === 0) {
              delete message.reactions[emoji];
            }
          }
        }
      } catch (error) {
        console.error('Failed to remove reaction:', error);
      }
    };

    const hasUserReactedWithEmoji = (message, emoji) => {
      return message.reactions?.[emoji]?.includes(currentUser.value.id) || false;
    };

    const hasUserReacted = (message) => {
      return Object.values(message.reactions || {}).some(users =>
          users.includes(currentUser.value.id)
      );
    };

    const getReactionUsers = (users) => {
      return users.map(userId => {
        if (userId === currentUser.value.id) return 'You';
        const user = store.getters.getUserById(userId);
        return user ? user.username : 'Unknown User';
      }).join(', ');
    };

    const getReplyPreview = (replyMessage) => {
      if (replyMessage.sender_id === currentUser.value.id) {
        return `You: ${replyMessage.content}`;
      }
      const sender = store.getters.getUserById(replyMessage.sender_id);
      return `${sender ? sender.username : 'Unknown User'}: ${replyMessage.content}`;
    };

    const replyToMessage = (message) => {
      store.commit('setReplyingTo', message);
    };

    const scrollToMessage = (messageId) => {
      const element = document.querySelector(`[data-message-id="${messageId}"]`);
      if (element) {
        element.scrollIntoView({ behavior: 'smooth', block: 'center' });
        element.classList.add('highlight');
        setTimeout(() => element.classList.remove('highlight'), 2000);
      }
    };

    const getMessageClass = (message) => {
      return message.sender_id === currentUser.value.id ? 'sent' : 'received';
    };

    const formatTime = (timestamp) => {
      if (!timestamp) return '';
      const date = new Date(timestamp);
      return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    };

    const getStatusIcon = (status) => {
      switch (status) {
        case 'sent': return '✓';
        case 'delivered': return '✓✓';
        case 'read': return '✓✓';
        default: return '';
      }
    };

    // *** NEW: Method to get the sender's username ***
    const getSenderUsername = (message) => {
      if (message.sender_username) { //Prioritize
        return message.sender_username
      }
      // Fallback to using usersOnline (less reliable)
      const sender = store.getters.getUserById(message.sender_id);
      return sender ? sender.username : 'Unknown User';
    };

    return {
      currentUser,
      showReactionPicker,
      selectedMessageId,
      hoveredMessageId,
      messagesContainer,
      availableReactions,
      toggleReactionPicker,
      closeReactionPicker,
      addReaction,
      removeReaction,
      getMessageClass,
      formatTime,
      getReplyPreview,
      hasUserReacted,
      hasUserReactedWithEmoji,
      handleReactionClick,
      getReactionUsers,
      replyToMessage,
      scrollToMessage,
      getStatusIcon,
      getSenderUsername, // Add to returned object
    };
  }
};
</script>

<style scoped>
/* Add style for sender username */
.message-sender {
  font-size: 0.8em;
  color: #888;
  margin-bottom: 2px;
}

/* ... (rest of your existing styles) ... */
.messages {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.message {
  max-width: 70%;
  padding: 8px 12px;
  border-radius: 8px;
  margin: 4px 0;
  position: relative;
}

.sent {
  align-self: flex-end;
  background-color: #007bff;
  color: white;
}

.received {
  align-self: flex-start;
  background-color: #e9ecef;
  color: black;
}

.message-content {
  position: relative;
}

.reply-preview {
  font-size: 0.8em;
  padding: 4px 8px;
  margin-bottom: 4px;
  background-color: rgba(0, 0, 0, 0.1);
  border-radius: 4px;
  cursor: pointer;
}

.reply-preview:hover {
  background-color: rgba(0, 0, 0, 0.2);
}

.message-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-top: 4px;
  opacity: 0.7;
  transition: opacity 0.2s;
  position: relative;
}
.message:hover .message-actions {
  opacity: 1;
}
.action-button {
  background: none;
  border: none;
  padding: 2px 6px;
  cursor: pointer;
  border-radius: 4px;
  transition: background-color 0.2s;
}

.action-button:hover {
  background-color: rgba(0, 0, 0, 0.1);
}
.action-buttons {
  display: flex;
  gap: 4px;
}
.reactions-container {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  align-items: center;
}

.reactions {
  display: flex;
  gap: 4px;
}

.reaction {
  background-color: rgba(255, 255, 255, 0.2);
  padding: 2px 4px;
  border-radius: 12px;
  font-size: 0.8em;
  cursor: pointer;
  transition: background-color 0.2s;
}

.reaction:hover {
  background-color: rgba(255, 255, 255, 0.3);
}

.user-reacted {
  background-color: rgba(255, 255, 255, 0.4);
  font-weight: bold;
}

.reaction-button {
  cursor: pointer;
  opacity: 0.6;
  transition: opacity 0.2s;
  padding: 2px 6px;
  font-size: 0.9em;
}

.reaction-button:hover {
  opacity: 1;
}

.reaction-picker {
  position: absolute;
  bottom: 100%; /* Position above the message */
  left: 0;
  background-color: white;
  border: 1px solid #ddd;
  border-radius: 8px;
  padding: 8px;
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  z-index: 1000; /* Make sure it appears above other elements */
  flex-wrap: wrap; /* Allow emojis to wrap to the next line */
  max-height: 200px; /* Set a maximum height */
  overflow-y: auto; /* Add scrollbar if needed */
}

.emoji-option {
  cursor: pointer;
  padding: 4px;
  border-radius: 4px;
  transition: background-color 0.2s;
  font-size: 1.2em; /* Make emojis bigger */
}

.emoji-option:hover {
  background-color: #f0f0f0;
}

.emoji-option.selected {
  background-color: #e0e0e0;
}


.message-info {
  font-size: 0.7em;
  margin-top: 2px;
  opacity: 0.8;
}

.highlight {
  animation: highlight 2s ease-out;
}

@keyframes highlight {
  0% {
    background-color: rgba;
    background-color: rgba(255, 255, 0, 0.5);
  }
  100% {
    background-color: transparent;
  }
}
.message-status {
  font-size: 0.8em;
  margin-left: 4px;
}

.message-status.sent {
  color: #999;
}

.message-status.delivered {
  color: #666;
}

.message-status.read {
  color: #0084ff;
}
.sent .action-button:hover {
  background-color: rgba(255, 255, 255, 0.2);
}
</style>
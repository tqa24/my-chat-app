<template>
  <div class="chat-messages">
    <div v-for="message in reversedMessages" :key="message.id" :class="messageClass(message)"
         @mouseenter="hoveredMessage = message.id"
         @mouseleave="hoveredMessage = null"
    >
      <div class="message-container">
        <div class="message-wrapper">
          <!-- Reply display -->
          <div v-if="message.reply_to_message" class="reply-message">
            <div class="reply-header">
              <span>Replying to {{ userIdToName[message.reply_to_message.sender_id] || 'Unknown User' }}</span>
            </div>
            <div class="reply-content">
              <span>{{ message.reply_to_message.content }}</span>
            </div>
          </div>

          <div class="message-content">
            <span>{{ message.content }}</span>
          </div>

          <!-- Message Options (appears on hover) -->
          <div class="message-options" v-show="hoveredMessage === message.id">
            <button class="option-btn" @click="replyToMessage(message)">
              <i class="fas fa-reply option-icon"></i>
            </button>
            <button class="option-btn" @click="showReactionPicker = !showReactionPicker; showMessageOptions = false">
              <i class="fas fa-smile option-icon"></i>
            </button>
            <button class="option-btn" @click="showMessageOptions = !showMessageOptions; showReactionPicker = false">
              <i class="fas fa-ellipsis-h option-icon"></i>
            </button>
          </div>

          <!-- Reaction Picker (Conditional) -->
          <div v-if="showReactionPicker && hoveredMessage === message.id" class="reaction-picker">
            <span @click="addReaction(message, '❤️')">❤️</span>
            <span @click="addReaction(message, '😂')">😂</span>
            <span @click="addReaction(message, '😮')">😮</span>
            <span @click="addReaction(message, '😢')">😢</span>
            <span @click="addReaction(message, '😠')">😠</span>
            <span @click="addReaction(message, '👍')">👍</span>
            <span @click="addReaction(message, '+')" class="add-reaction-plus">+</span>
          </div>

          <!-- Display existing reactions -->
          <div class="reactions" v-show="hasReactions(message)">
            <span v-for="(reaction, emoji) in message.reactions" :key="emoji" class="reaction" @click="toggleReaction(message, emoji)">
              {{ emoji }} {{ reaction.length }}
            </span>
          </div>
        </div>
      </div>
      <div class="message-meta">
        {{ formatTime(message.created_at) }}
        <span> - </span>
        {{ message.sender_id == currentUser?.id ? currentUser.username : userIdToName[message.sender_id] }}
        <span> - </span>
        {{ message.status }}
      </div>
    </div>
  </div>
</template>

<script>
import { computed, ref } from 'vue';
import { useStore } from 'vuex';
import { format } from 'date-fns';
import axios from 'axios';

export default {
  setup() {
    const store = useStore();
    const messages = computed(() => store.getters.allMessages);
    const currentUser = computed(() => store.getters.currentUser);
    const usersOnline = computed(() => store.getters.getUsersOnline);
    const reversedMessages = computed(() => [...messages.value].reverse());
    const userIdToName = computed(() => {
      const map = {};
      usersOnline.value.forEach(user => {
        map[user.id] = user.username;
      });
      return map;
    });

    // State for hover and reaction picker
    const hoveredMessage = ref(null);
    const showReactionPicker = ref(false);
    const showMessageOptions = ref(false);

    const messageClass = (message) => ({
      'message': true,
      'sent-message': currentUser.value?.id === message.sender_id,
      'received-message': currentUser.value?.id !== message.sender_id,
    });

    const formatTime = (timestamp) => format(new Date(timestamp), 'HH:mm');

    const addReaction = async (message, reaction) => {
      try {
        const userReacted = message.reactions[reaction]?.includes(currentUser.value.id);
        const shouldAdd = !userReacted; // Calculate *before* the optimistic update

        // Optimistically update the UI *before* sending the request.
        store.dispatch('toggleReaction', { messageId: message.id, reaction, add: shouldAdd });

        if (!shouldAdd) { // Use the pre-calculated value
          // Remove reaction
          await axios.delete(`http://localhost:8080/messages/${message.id}/react`, {
            data: { user_id: currentUser.value.id, reaction: reaction }
          });
        } else {
          // Add reaction
          await axios.post(`http://localhost:8080/messages/${message.id}/react`, {
            user_id: currentUser.value.id,
            reaction: reaction,
          });
        }

        // No need to fetch messages here; optimistic update handles it

      } catch (error) {
        console.error('Failed to toggle reaction:', error);
        // If the request fails, revert the optimistic update
        const userReacted = message.reactions[reaction]?.includes(currentUser.value.id); //re-calculate
        const shouldAdd = !userReacted;//re-calculate
        store.dispatch('toggleReaction', { messageId: message.id, reaction, add: !shouldAdd }); // Revert: Use the OPPOSITE of shouldAdd
      }
    };

    const replyToMessage = (message) => {
      store.commit('setReplyingTo', message);
    };
    //Helper function
    const hasReactions = (message) => {
      return message.reactions && Object.keys(message.reactions).length > 0;
    }

    return {
      reversedMessages, messages, currentUser, messageClass, formatTime,
      userIdToName, replyToMessage, hoveredMessage,
      showReactionPicker, showMessageOptions, hasReactions, addReaction
    };
  }
};
</script>

<style scoped>
.chat-messages {
  padding: 10px;
  overflow-y: auto;
  height: 300px;
  display: flex;
  flex-direction: column-reverse;
}

.message-container {
  display: flex;
  align-items: flex-start;
  margin-bottom: 10px;
}

.message-wrapper {
  position: relative; /* Important for positioning children */
  max-width: 70%; /* Limit message width */
}

.message {
  padding: 8px 12px;
  border-radius: 18px;
  word-wrap: break-word;
  margin-bottom: 2px; /* Space between message content and reactions */
}

.sent-message {
  background-color: #0084ff;
  color: white;
  align-self: flex-end;
  margin-left: auto; /* Push to the right */
  border-top-right-radius: 2px; /*  Less rounded corner */
}

.received-message {
  background-color: #f0f0f0;
  color: black;
  align-self: flex-start;
  margin-right: auto; /* Push to the left */
  border-top-left-radius: 2px;  /* Less rounded corner */
}

.message-meta {
  font-size: 11px;
  color: #888;
  margin-top: 4px;
  clear: both; /* Clear any floats from absolute positioning */
}

/* Reply Message Styles */
.reply-message {
  background-color: rgba(0, 0, 0, 0.05); /* Lighter background */
  padding: 5px 8px;
  margin-bottom: 4px;
  border-radius: 4px;
  border-left: 3px solid #0084ff;
  font-size: smaller;
}

.reply-header {
  font-weight: bold;
  margin-bottom: 2px;
}

.reply-content {
  font-style: italic;
  color: #555;
}

/* Message Options (Hidden by default) */
.message-options {
  position: absolute;
  top: -5px; /* Position slightly above the message */
  right: 5px;  /* Position to the right */
  display: none; /* Hidden by default */
  background-color: white;
  border: 1px solid #ccc;
  border-radius: 4px;
  padding: 2px;
  z-index: 10;
  box-shadow: 0 1px 3px rgba(0,0,0,0.2); /* Add a shadow */
}

/* Show options on hover */
.message-wrapper:hover .message-options {
  display: flex;
}

.option-btn {
  background: none;
  border: none;
  padding: 2px 5px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
}

.option-icon {
  font-size: 18px; /* Slightly larger icons */
}

/* Reaction Picker Styles */
.reaction-picker {
  position: absolute;
  bottom: 25px;     /* Position above the message options */
  right: 0px;
  display: flex;
  background-color: white;
  border: 1px solid #ccc;
  border-radius: 15px; /* Rounded corners */
  padding: 5px;
  z-index: 20;      /* Above message options */
  box-shadow: 0 2px 4px rgba(0,0,0,0.2); /* Add a subtle shadow */
}

.reaction-picker span {
  cursor: pointer;
  margin: 0 5px; /* More spacing */
  font-size: 20px; /*  larger emojis */
  line-height: 1;
}
/* Style the "+" in the reaction picker */
.add-reaction-plus{
  color: #888;
}

/* Reactions Display */
.reactions {
  position: absolute;
  bottom: -20px;
  left: 0;
  display: flex;
  z-index: 5; /* Below options and picker, but above message */
}

.reaction {
  margin-right: 4px;
  font-size: 11px;
  background-color: rgba(255, 255, 255, 0.9); /* Semi-transparent white */
  border: 1px solid #ccc;
  border-radius: 10px;
  padding: 1px 4px;
  line-height: 1;
  box-shadow: 0 1px 2px rgba(0,0,0,0.1); /* Subtle shadow */
  cursor: pointer; /* Add cursor pointer */
}
</style>
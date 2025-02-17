<template>
  <div class="chat-input">
    <div v-if="replyingTo" class="reply-preview">
      <span>Replying to {{ replyingTo.sender_id == currentUser?.id ? currentUser.username : userIdToName[replyingTo.sender_id] }} :</span>
      <span>{{ replyingTo.content }}</span>
      <button @click="cancelReply">Cancel</button>
    </div>
    <FileUpload @file-uploaded="handleFileUploaded" @file-removed="removeFile" />
    <div class="input-container">
      <textarea
          v-model="message"
          @keydown="handleKeydown"
          @input="handleTyping"
          placeholder="Type your message..."
          ref="messageInput"
      ></textarea>
      <!-- Suggestion popup -->
      <div v-if="showSuggestions" class="suggestions-popup">
        <div
            class="suggestion-item"
            :class="{ active: true }"
            @click="selectSuggestion"
        >
          @AI
        </div>
      </div>
    </div>
    <button @click="sendMessage">Send</button>
  </div>
</template>

<script>
import { ref, computed, watch, onUnmounted, nextTick } from 'vue';
import { useStore } from 'vuex';
import FileUpload from './FileUpload.vue';

export default {
  components: {
    FileUpload,
  },
  props:{
    receiverID: {
      type: String,
      required: false,
      default: null,
    },
    groupID: {
      type: String,
      required: false,
      default: null,
    }
  },
  setup(props) {
    const store = useStore();
    const message = ref('');
    const messageInput = ref(null);
    const showSuggestions = ref(false);
    const currentWordStart = ref(0);
    const currentWordEnd = ref(0);
    const currentUser = computed(() => store.getters.currentUser);
    const replyingTo = computed(() => store.state.replyingTo);
    const userIdToName = computed(() => {
      const map = {};
      store.state.usersOnline.forEach(user => {
        map[user.id] = user.username;
      });
      return map;
    });

    // File upload state
    const uploadedFile = ref(null);

    // Typing indicator variables
    let typingTimeout = null;

    const draftKey = computed(() => {
      if (props.groupID) {
        return `draft_group_${props.groupID}`;
      } else if (props.receiverID) {
        return `draft_user_${props.receiverID}`;
      }
      return null;
    });

    watch(draftKey, (newKey, oldKey) => {
      if (oldKey) {
        localStorage.setItem(oldKey, message.value);
      }
      if (newKey) {
        message.value = localStorage.getItem(newKey) || '';
      }
    });

    onUnmounted(() => {
      if (draftKey.value) {
        localStorage.setItem(draftKey.value, message.value);
      }
      clearTimeout(typingTimeout);
    });

    const handleFileUploaded = (fileInfo) => {
      uploadedFile.value = fileInfo;
    };

    const removeFile = () => {
      uploadedFile.value = null;
    };

    const handleKeydown = (e) => {
      if (showSuggestions.value) {
        if (e.key === 'Tab' || e.key === 'Enter') {
          e.preventDefault();
          selectSuggestion();
        }
        if (e.key === 'Escape') {
          showSuggestions.value = false;
        }
      } else if (e.key === '@') {
        showSuggestions.value = true;
        currentWordStart.value = messageInput.value.selectionStart;
      } else if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        sendMessage();
      }
    };

    const handleTyping = () => {
      // Get cursor position
      const cursorPosition = messageInput.value.selectionStart;

      // Check if we're in an @ mention context
      const textBeforeCursor = message.value.substring(0, cursorPosition);
      const lastAtSymbol = textBeforeCursor.lastIndexOf('@');

      if (lastAtSymbol !== -1) {
        const wordAfterAt = textBeforeCursor.substring(lastAtSymbol + 1);
        // Show suggestions if we're right after @ or if "AI" starts with the typed text
        if (wordAfterAt === '' || 'AI'.toLowerCase().startsWith(wordAfterAt.toLowerCase())) {
          showSuggestions.value = true;
          currentWordStart.value = lastAtSymbol;
          currentWordEnd.value = cursorPosition;
        } else {
          showSuggestions.value = false;
        }
      } else {
        showSuggestions.value = false;
      }

      // Handle typing indicator
      clearTimeout(typingTimeout);

      if (draftKey.value) {
        localStorage.setItem(draftKey.value, message.value);
      }

      if ((props.groupID || props.receiverID) && store.state.ws) {
        let typingMsg = {};
        if (props.groupID) {
          typingMsg = {
            type: "typing",
            sender_id: currentUser.value.id,
            group_id: props.groupID,
          };
        } else {
          typingMsg = {
            type: "typing",
            sender_id: currentUser.value.id,
            receiver_id: props.receiverID,
          };
        }

        store.state.ws.send(JSON.stringify(typingMsg));

        typingTimeout = setTimeout(() => {
          let stopTypingMsg = {};

          if (props.groupID) {
            stopTypingMsg = {
              type: "stop_typing",
              sender_id: currentUser.value.id,
              group_id: props.groupID
            };
          } else {
            stopTypingMsg = {
              type: "stop_typing",
              sender_id: currentUser.value.id,
              receiver_id: props.receiverID
            };
          }

          if (store.state.ws) {
            store.state.ws.send(JSON.stringify(stopTypingMsg));
          }
        }, 2000);
      }
    };

    const selectSuggestion = () => {
      const beforeMention = message.value.substring(0, currentWordStart.value);
      const afterMention = message.value.substring(currentWordEnd.value);
      message.value = beforeMention + '@AI ' + afterMention;
      showSuggestions.value = false;

      // Set cursor position after the inserted mention
      nextTick(() => {
        const newPosition = currentWordStart.value + 4; // '@AI '.length = 4
        messageInput.value.setSelectionRange(newPosition, newPosition);
        messageInput.value.focus();
      });
    };

    const sendMessage = () => {
      if (message.value.trim() !== '' || uploadedFile.value) {
        let msg = {};
        if (props.groupID) {
          msg = {
            type: "new_message",
            sender_id: currentUser.value.id,
            group_id: props.groupID,
            content: message.value,
            reply_to_message_id: replyingTo.value ? replyingTo.value.id : null,
            file_name: uploadedFile.value ? uploadedFile.value.name : null,
            file_path: uploadedFile.value ? uploadedFile.value.path : null,
            file_type: uploadedFile.value ? uploadedFile.value.type : null,
            file_size: uploadedFile.value ? uploadedFile.value.size : null,
            checksum: uploadedFile.value ? uploadedFile.value.checksum : null,
          };
        } else {
          msg = {
            type: "new_message",
            sender_id: currentUser.value.id,
            receiver_id: props.receiverID,
            content: message.value,
            reply_to_message_id: replyingTo.value ? replyingTo.value.id : null,
            file_name: uploadedFile.value ? uploadedFile.value.name : null,
            file_path: uploadedFile.value ? uploadedFile.value.path : null,
            file_type: uploadedFile.value ? uploadedFile.value.type : null,
            file_size: uploadedFile.value ? uploadedFile.value.size : null,
            checksum: uploadedFile.value ? uploadedFile.value.checksum : null,
          };
        }

        if (store.state.ws) {
          store.state.ws.send(JSON.stringify(msg));
        } else {
          console.error("WebSocket connection is not available.");
        }

        message.value = '';
        store.commit('setReplyingTo', null);
        uploadedFile.value = null;
      }
      if (draftKey.value) {
        localStorage.removeItem(draftKey.value);
      }
    };

    const cancelReply = () => {
      store.commit('setReplyingTo', null);
    };

    return {
      message,
      messageInput,
      showSuggestions,
      currentUser,
      replyingTo,
      userIdToName,
      handleKeydown,
      handleTyping,
      selectSuggestion,
      sendMessage,
      handleFileUploaded,
      removeFile,
      cancelReply
    };
  }
};
</script>

<style scoped>
.chat-input {
  display: flex;
  flex-direction: column;
  padding: 10px;
  border-top: 1px solid #ccc;
}

.reply-preview {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background-color: #f0f0f0;
  padding: 5px;
  margin-bottom: 5px;
  border-radius: 4px;
}

.reply-preview > span {
  margin-right: 5px;
}

.reply-preview button {
  background: none;
  border: none;
  color: red;
  cursor: pointer;
}

.input-container {
  position: relative;
  flex-grow: 1;
}

textarea {
  width: 100%;
  min-height: 60px;
  padding: 8px;
  border: 1px solid #ccc;
  border-radius: 4px;
  resize: vertical;
}

.suggestions-popup {
  position: absolute;
  bottom: 100%;
  left: 0;
  background: white;
  border: 1px solid #ccc;
  border-radius: 4px;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
  z-index: 1000;
  margin-bottom: 4px;
}

.suggestion-item {
  padding: 8px 12px;
  cursor: pointer;
  transition: background-color 0.2s;
}

.suggestion-item:hover,
.suggestion-item.active {
  background-color: #f0f0f0;
}

button {
  background-color: #4CAF50;
  color: white;
  padding: 8px 15px;
  border: none;
  border-radius: 5px;
  cursor: pointer;
  margin-top: 8px;
}

button:hover {
  background-color: #45a049;
}
</style>
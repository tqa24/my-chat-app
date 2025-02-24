<template>
  <div class="chat-input">
    <div v-if="replyingTo" class="reply-preview">
      <span>Replying to {{ replyingTo.sender_id == currentUser?.id ? currentUser.username : userIdToName[replyingTo.sender_id] }} :</span>
      <span>{{ replyingTo.content }}</span>
      <button @click="cancelReply">Cancel</button>
    </div>
    <FileUpload @file-uploaded="handleFileUploaded" @file-removed="removeFile" />
    <div class="textarea-container">
      <textarea
          ref="textarea"
          v-model="message"
          @keydown="handleKeyDown"
          @input="handleTyping"
          placeholder="Type your message..."
      ></textarea>
      <div v-if="showSuggestions" class="suggestions-dropdown">
        <div v-for="(suggestion, index) in suggestions" :key="index"
             @click="selectSuggestion(suggestion)"
             :class="{ 'selected': selectedSuggestionIndex === index }"
        >
          {{ suggestion }}
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
    const currentUser = computed(()=> store.getters.currentUser)
    const replyingTo = computed(() => store.state.replyingTo);
    const textarea = ref(null);  // Ref for the textarea element
    const showSuggestions = ref(false);
    const suggestions = ref(['@AI']); // Start with @AI
    const selectedSuggestionIndex = ref(-1); // Track selected suggestion

    const userIdToName = computed(() => {
      const map = {};
      store.state.usersOnline.forEach(user => {
        map[user.id] = user.username;
      });
      return map;
    });


    // *** File upload state ***
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


    // Clear the timeout when the component is unmounted
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
    }


    const sendMessage = () => {
      if (message.value.trim() !== '' || uploadedFile.value) { // Send even if only file
        let msg = {};
        if(props.groupID){
          msg = {
            type: "new_message",
            sender_id: currentUser.value.id,
            group_id: props.groupID,
            content: message.value,
            reply_to_message_id: replyingTo.value ? replyingTo.value.id : null,
            // *** Add file information ***
            file_name: uploadedFile.value ? uploadedFile.value.name : null,
            file_path: uploadedFile.value ? uploadedFile.value.path : null,
            file_type: uploadedFile.value ? uploadedFile.value.type : null,
            file_size: uploadedFile.value ? uploadedFile.value.size : null,
            checksum:  uploadedFile.value ? uploadedFile.value.checksum : null, // Pass checksum
          }
        } else {
          msg = {
            type: "new_message",
            sender_id: currentUser.value.id,
            receiver_id: props.receiverID,
            content: message.value,
            reply_to_message_id: replyingTo.value ? replyingTo.value.id : null,
            // *** Add file information ***
            file_name: uploadedFile.value ? uploadedFile.value.name : null,
            file_path: uploadedFile.value ? uploadedFile.value.path : null,
            file_type: uploadedFile.value ? uploadedFile.value.type : null,
            file_size: uploadedFile.value ? uploadedFile.value.size : null,
            checksum: uploadedFile.value ? uploadedFile.value.checksum : null, // Pass checksum
          }
        }

        // Check if the WebSocket connection exists before sending
        console.log("Sending message:", msg);
        if (store.state.ws) {
          store.state.ws.send(JSON.stringify(msg));
        } else {
          console.error("WebSocket connection is not available.");
          // Consider showing an error message to the user or attempting to reconnect
        }

        message.value = '';
        store.commit('setReplyingTo', null);
        uploadedFile.value = null; // Clear the file after sending
        showSuggestions.value = false; // Hide suggestions
      }
      if(draftKey.value){
        localStorage.removeItem(draftKey.value)
      }
    };

    const handleKeyDown = (event) => {
      if (event.key === 'Enter' && !event.shiftKey && !showSuggestions.value) {
        event.preventDefault();
        sendMessage();
      }

      if (event.key === '@') {
        showSuggestions.value = true;
        selectedSuggestionIndex.value = 0; // 0 => select first suggestion by default
        nextTick(() => positionDropdown());
      } else if (event.key === 'Escape') {
        showSuggestions.value = false;
      } else if (showSuggestions.value) {
        // Handle arrow key navigation
        if (event.key === 'ArrowUp') {
          event.preventDefault(); // Prevent cursor moving
          selectedSuggestionIndex.value = Math.max(0, selectedSuggestionIndex.value - 1);
        } else if (event.key === 'ArrowDown') {
          event.preventDefault();
          selectedSuggestionIndex.value = Math.min(suggestions.value.length - 1, selectedSuggestionIndex.value + 1);
        } else if (event.key === 'Enter' || event.key === 'Tab') {
          event.preventDefault();
          if (suggestions.value.length > 0) {
            selectSuggestion(suggestions.value[selectedSuggestionIndex.value]);
          }
        }
      } else {
        showSuggestions.value = false;
      }
    };

    const selectSuggestion = (suggestion) => {
      const cursorPosition = textarea.value.selectionStart;
      const textBeforeCursor = message.value.substring(0, cursorPosition);
      const textAfterCursor = message.value.substring(cursorPosition);

      // Find the last occurrence of '@' before the cursor
      const lastAtIndex = textBeforeCursor.lastIndexOf('@');

      if (lastAtIndex !== -1) {
        // Replace the text from the last '@' to the cursor with the suggestion
        message.value = textBeforeCursor.substring(0, lastAtIndex) + suggestion + " " + textAfterCursor;
      } else {
        message.value = suggestion + " " + textAfterCursor
      }

      showSuggestions.value = false;
      nextTick(() => {
        textarea.value.focus();
        // Set the cursor position after the inserted suggestion
        textarea.value.selectionStart = textarea.value.selectionEnd = lastAtIndex + suggestion.length + 1;
      });
    };

    const positionDropdown = () => {
      // No need for complex positioning since we're using top: 100%
      const dropdown = document.querySelector('.suggestions-dropdown');
      if (dropdown) {
        dropdown.style.display = 'block';
      }
    }



    const handleTyping = () => {
      clearTimeout(typingTimeout);

      // Check typing with @
      const cursorPosition = textarea.value.selectionStart;
      const textBeforeCursor = message.value.substring(0, cursorPosition);
      const lastAtIndex = textBeforeCursor.lastIndexOf('@');

      if (lastAtIndex !== -1) {
        const searchTerm = textBeforeCursor.substring(lastAtIndex + 1).toLowerCase(); // Add toLowerCase()

        // Show @AI suggestion when typing @ or @a or @ai (case insensitive)
        if (searchTerm === '' || 'ai'.startsWith(searchTerm)) {
          suggestions.value = ['@AI'];
          showSuggestions.value = true;
          nextTick(() => positionDropdown());
        } else {
          showSuggestions.value = false;
        }
      } else {
        showSuggestions.value = false;
      }

      if(draftKey.value){
        localStorage.setItem(draftKey.value, message.value)
      }

      if((props.groupID || props.receiverID) && store.state.ws){ // Check ws exists
        let typingMsg = {};
        if(props.groupID){
          typingMsg = {
            type: "typing",
            sender_id: currentUser.value.id,
            group_id: props.groupID,
          }
        } else {
          typingMsg = {
            type: "typing",
            sender_id: currentUser.value.id,
            receiver_id: props.receiverID,
          }
        }

        store.state.ws.send(JSON.stringify(typingMsg));

        typingTimeout = setTimeout(() => {
          let stopTypingMsg = {};

          if(props.groupID){
            stopTypingMsg = {
              type: "stop_typing",
              sender_id: currentUser.value.id,
              group_id: props.groupID
            }
          } else {
            stopTypingMsg = {
              type: "stop_typing",
              sender_id: currentUser.value.id,
              receiver_id: props.receiverID
            }
          }

          if (store.state.ws) { // Check ws before sending stop_typing
            store.state.ws.send(JSON.stringify(stopTypingMsg));
          }
        }, 2000);
      }
    };


    const cancelReply = () => {
      store.commit('setReplyingTo', null);
    }

    return {
      message,
      sendMessage,
      handleTyping,
      replyingTo,
      cancelReply,
      userIdToName,
      handleFileUploaded,
      removeFile,
      handleKeyDown,         // Add the handler
      showSuggestions,
      suggestions,
      selectedSuggestionIndex,
      selectSuggestion,
      textarea,
    };
  }
};
</script>

<style scoped>
/* (Your existing styles - no changes needed here) */
.chat-input {
  display: flex;
  flex-direction: column; /* Stack elements vertically */
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
.reply-preview > span{
  margin-right: 5px;
}

.reply-preview button {
  background: none;
  border: none;
  color: red;
  cursor: pointer;
}
.textarea-container {
  position: relative; /* For absolute positioning of dropdown */
  width: 100%;
}

textarea {
  flex-grow: 1;
  margin-bottom: 5px; /* Space between textarea and button */
  padding: 8px;
  border: 1px solid #ccc;
  border-radius: 5px;
  resize: none; /* Prevent textarea resizing */
  width: 100%;
  box-sizing: border-box;
}
.input-container{
  display: flex;
}

button {
  background-color: #4CAF50;
  color: white;
  padding: 8px 15px;
  border: none;
  border-radius: 5px;
  cursor: pointer;
}
.suggestions-dropdown {
  position: absolute;
  top: 100%; /* Change this from the current positioning */
  left: 0;
  z-index: 1000; /* Increase z-index to ensure visibility */
  background-color: white;
  border: 1px solid #ccc;
  border-radius: 4px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  max-height: 150px;
  overflow-y: auto;
  width: 100%;
}
.suggestions-dropdown div {
  padding: 8px 12px;
  cursor: pointer;
}
.suggestions-dropdown div:hover, .suggestions-dropdown div.selected {
  background-color: #f0f0f0;
}
</style>
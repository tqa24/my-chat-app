<template>
  <div class="chat-input">
    <div v-if="replyingTo" class="reply-preview">
      <span>Replying to {{ replyingTo.sender_id == currentUser?.id ? currentUser.username : userIdToName[replyingTo.sender_id] }} :</span>
      <span>{{ replyingTo.content }}</span>
      <button @click="cancelReply">Cancel</button>
    </div>
    <FileUpload @file-uploaded="handleFileUploaded" @file-removed="removeFile" />
    <textarea
        v-model="message"
        @keydown.enter.prevent="sendMessage"
        @input="handleTyping"
        placeholder="Type your message..."
    ></textarea>
    <button @click="sendMessage">Send</button>
  </div>
</template>

<script>
import { ref, computed, watch, onUnmounted } from 'vue';
import { useStore } from 'vuex';
import FileUpload from './FileUpload.vue'; // Import the new component

export default {
  components: {
    FileUpload, // Register FileUpload
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
    const userIdToName = computed(() => {
      const map = {};
      store.state.usersOnline.forEach(user => {
        map[user.id] = user.username;
      });
      return map;
    });

    // *** NEW: File upload state ***
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
      }
      if(draftKey.value){
        localStorage.removeItem(draftKey.value)
      }
    };

    const handleTyping = () => {
      clearTimeout(typingTimeout); // Clear existing timeout

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

    return { message, sendMessage, handleTyping, replyingTo, cancelReply, userIdToName, handleFileUploaded, removeFile};
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

textarea {
  flex-grow: 1;
  margin-bottom: 5px; /* Space between textarea and button */
  padding: 8px;
  border: 1px solid #ccc;
  border-radius: 5px;
  resize: none; /* Prevent textarea resizing */
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
</style>
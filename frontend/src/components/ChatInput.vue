<template>
  <div class="chat-input">
    <div v-if="replyingTo" class="reply-preview">
      <span>Replying to {{ replyingTo.sender_id == currentUser?.id ? currentUser.username : userIdToName[replyingTo.sender_id] }} :</span>
      <span>{{ replyingTo.content }}</span>
      <button @click="cancelReply">Cancel</button>
    </div>
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
import { ref, computed, watch, onUnmounted } from 'vue'; // Add computed here
import { useStore } from 'vuex';
let typingTimeout = null;
export default {
  props:{
    receiverID: {
      type: String,
      required: false, // Now optional
      default: null,
    },
    groupID: { // Add groupID prop
      type: String,
      required: false, // Optional
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
      store.state.usersOnline.forEach(user => { // Use store.state.usersOnline
        map[user.id] = user.username;
      });
      return map;
    });

    // Load draft from localStorage on component mount
    const draftKey = computed(() => {
      if (props.groupID) {
        return `draft_group_${props.groupID}`;
      } else if (props.receiverID) {
        return `draft_user_${props.receiverID}`;
      }
      return null; // No draft if no recipient/group
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
      // Save draft when component is unmounted
      if (draftKey.value) {
        localStorage.setItem(draftKey.value, message.value);
      }
    });

    const sendMessage = () => {
      if (message.value.trim() !== '') {
        let msg = {};
        // Determine if it is a group message or direct message
        if(props.groupID){
          msg = {
            type: "new_message",
            sender_id: currentUser.value.id,
            group_id: props.groupID, // Send group_id if set
            content: message.value,
            reply_to_message_id: replyingTo.value ? replyingTo.value.id : null, // Add reply_to_message_id

          }
        } else {
          msg = {
            type: "new_message",
            sender_id: currentUser.value.id,
            receiver_id: props.receiverID, // Send receiver_id if set
            content: message.value,
            reply_to_message_id: replyingTo.value ? replyingTo.value.id : null, // Add reply_to_message_id
          }
        }
        store.state.ws.send(JSON.stringify(msg)) // Send over WebSocket

        message.value = ''; // Clear input after sending
        // Clear reply after sending
        store.commit('setReplyingTo', null);
      }
      // Remove draft from local storage
      if(draftKey.value){
        localStorage.removeItem(draftKey.value)
      }
    };
    const handleTyping = () => {
      // Clear any existing timeout
      clearTimeout(typingTimeout);

      // Save draft to localStorage
      if(draftKey.value){
        localStorage.setItem(draftKey.value, message.value)
      }


      if(props.groupID || props.receiverID){
        // Send "typing" event immediately
        let typingMsg = {};
        if(props.groupID){
          typingMsg = {
            type: "typing",
            sender_id: currentUser.value.id,
            group_id: props.groupID, // Send group_id if set

          }
        } else {
          typingMsg = {
            type: "typing",
            sender_id: currentUser.value.id,
            receiver_id: props.receiverID, // Send receiver_id if set

          }
        }
        store.state.ws.send(JSON.stringify(typingMsg));
        // Set a timeout to send "stop_typing" after a delay (e.g., 2 seconds)
        typingTimeout = setTimeout(() => {
          let stopTypingMsg = {};
          if(props.groupID) {
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
          store.state.ws.send(JSON.stringify(stopTypingMsg));
        }, 2000); // 2 seconds
      }

    };

    const cancelReply = () => {
      store.commit('setReplyingTo', null);
    }

    return { message, sendMessage, handleTyping, replyingTo, cancelReply, userIdToName }; // Return replyingTo and cancelReply
  }
};
</script>

<style scoped>
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
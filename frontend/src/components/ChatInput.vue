<template>
  <div class="chat-input">
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
import { ref, computed } from 'vue'; // Add computed here
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

    const sendMessage = () => {
      if (message.value.trim() !== '') {
        let msg = {};
        // Determine if it is a group message or direct message
        if(props.groupID){
          msg = {
            type: "new_message",
            sender_id: currentUser.value.id,
            group_id: props.groupID, // Send group_id if set
            content: message.value
          }
        } else {
          msg = {
            type: "new_message",
            sender_id: currentUser.value.id,
            receiver_id: props.receiverID, // Send receiver_id if set
            content: message.value
          }
        }
        store.state.ws.send(JSON.stringify(msg)) // Send over WebSocket

        message.value = ''; // Clear input after sending
      }
    };
    const handleTyping = () => {
      // Clear any existing timeout
      clearTimeout(typingTimeout);
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

    return { message, sendMessage, handleTyping };
  }
};
</script>

<style scoped>
.chat-input {
  display: flex;
  padding: 10px;
  border-top: 1px solid #ccc;
}

textarea {
  flex-grow: 1;
  margin-right: 10px;
  padding: 8px;
  border: 1px solid #ccc;
  border-radius: 5px;
  resize: none; /* Prevent textarea resizing */
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
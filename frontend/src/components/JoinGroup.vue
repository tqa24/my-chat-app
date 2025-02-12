<template>
  <div class="join-group-container">
    <h2>Join Group by Code</h2>
    <form @submit.prevent="joinGroup">
      <div class="form-group">
        <label for="groupCode">Group Code:</label>
        <input type="text" id="groupCode" v-model="groupCode" required>
      </div>
      <button type="submit">Join Group</button>
      <p v-if="error" class="error-message">{{ error }}</p>
    </form>
    <!-- Display the group code after successful join -->
    <div v-if="joinedGroup" class="joined-group">
      <p>Successfully joined group: <strong>{{ joinedGroup.name }}</strong></p>
      <p>Group Code: <strong>{{ joinedGroup.code }}</strong></p>
      <button @click="copyCode(joinedGroup.code)">Copy Code</button>
      <p v-if="copyMessage">{{ copyMessage }}</p>
    </div>
  </div>
</template>

<script>
import axios from 'axios';
import { ref } from 'vue';
import { useRouter } from 'vue-router';
import {useStore} from 'vuex';
export default {
  name: 'JoinGroup',
  setup() {
    const groupCode = ref('');
    const error = ref('');
    const router = useRouter();
    const store = useStore();
    const joinedGroup = ref(null); // To store group data after joining
    const copyMessage = ref("");
    const copyCode = (code) => {
      navigator.clipboard.writeText(code)
          .then(() => {
            copyMessage.value = "Code copied to clipboard!";
            setTimeout(() => { copyMessage.value = "" }, 2000);
          })
          .catch((err) => {
            console.error("Failed to copy code: ", err);
          });
    };
    const joinGroup = async () => {
      joinedGroup.value = null; // Reset
      error.value = "";       // Reset
      try {
        const currentUser = store.getters.currentUser;
        if (!currentUser || !currentUser.id) {
          error.value = "User not logged in.";
          return;
        }
        const response = await axios.post('http://localhost:8080/groups/join-by-code', {
          code: groupCode.value, // Send the code
          user_id: currentUser.id,   // Temporary workaround
        });
        //Success
        if(response.status === 200 && response.data.group){
          joinedGroup.value = response.data.group; // Store group info
          groupCode.value = ''; // Clear input
        } else {
          error.value = 'Failed to join group: Invalid data';
        }


        // router.push('/'); // Redirect to the home page. We'll change this
        // console.log("Join Group response: ", response);
      } catch (err) {
        error.value = err.response?.data?.error || 'Failed to join group';
        console.log(error.value);
      }
    };

    return {
      groupCode,
      error,
      joinGroup,
      joinedGroup, // Make joinedGroup available
      copyCode,
      copyMessage
    };
  },
};
</script>

<style scoped>
/* Same styles as CreateGroup - consider putting in a shared CSS file */
.join-group-container {
  width: 300px;
  margin: 20px auto;
  padding: 20px;
  border: 1px solid #ccc;
  border-radius: 5px;
}
.form-group {
  margin-bottom: 15px;
}
label {
  display: block;
  margin-bottom: 5px;
}
input[type="text"] {
  width: 100%;
  padding: 8px;
  border: 1px solid #ccc;
  border-radius: 4px;
  box-sizing: border-box;
}
button {
  background-color: #4CAF50;
  color: white;
  padding: 10px 15px;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}
button:hover {
  background-color: #45a049;
}
.error-message {
  color: red;
  margin-top: 10px;
}
/* Style for displaying joined group info */
.joined-group {
  margin-top: 20px;
  padding: 10px;
  border: 1px solid #ddd;
  background-color: #f9f9f9;
}
.copy-message {
  margin-left: 10px;
  color: green;
}
</style>
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

    const joinGroup = async () => {
      try {
        const currentUser = store.getters.currentUser;
        if (!currentUser || !currentUser.id) {
          error.value = "User not logged in.";
          return;
        }
        const response = await instance.post('/groups/join-by-code', {
          code: groupCode.value, // Send the code
          user_id: currentUser.id,   // Temporary workaround
        });

        router.push('/'); // Redirect to the home page
        console.log("Join Group response: ", response);
      } catch (err) {
        error.value = err.response?.data?.error || 'Failed to join group';
      }
    };

    const instance = axios.create({
      baseURL: '/api', // Set base URL for all axios requests
    });

    return {
      groupCode,
      error,
      joinGroup,
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
</style>
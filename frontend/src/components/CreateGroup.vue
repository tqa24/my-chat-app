<template>
  <div class="create-group-container">
    <h2>Create New Group</h2>
    <form @submit.prevent="createGroup">
      <div class="form-group">
        <label for="groupName">Group Name:</label>
        <input type="text" id="groupName" v-model="groupName" required>
      </div>
      <button type="submit">Create Group</button>
      <p v-if="error" class="error-message">{{ error }}</p>
    </form>
  </div>
</template>

<script>
import axios from 'axios';
import { ref } from 'vue';
import { useRouter } from 'vue-router';
import { useStore } from 'vuex';

export default {
  name: 'CreateGroup',
  setup() {
    const groupName = ref('');
    const error = ref('');
    const router = useRouter();
    const store = useStore();

    const createGroup = async () => {
      try {
        const currentUser = store.getters.currentUser;

        // Check if currentUser and currentUser.id exist
        if (!currentUser || !currentUser.id) {
          error.value = 'User not logged in.'; // Or redirect to login
          return;
        }
        const response = await axios.post('http://localhost:8080/groups', {
          name: groupName.value,
          creator_id: currentUser.id // Send the creator_id
        });

        // Group created successfully, redirect to home page
        router.push('/');
        console.log("New Group response ", response)
      } catch (err) {
        error.value = err.response?.data?.error || 'Failed to create group';
      }
    };

    return {
      groupName,
      error,
      createGroup,
    };
  },
};
</script>

<style scoped>
.create-group-container {
  width: 300px;
  margin: 0 auto; /* Center the form */
  padding: 20px;
  border: 1px solid #ccc;
  border-radius: 5px;
}

.form-group {
  margin-bottom: 15px;
}

label {
  display: block; /* Make labels block-level */
  margin-bottom: 5px;
}

input[type="text"],
input[type="password"],
input[type="email"] {
  width: 100%;
  padding: 8px;
  border: 1px solid #ccc;
  border-radius: 4px;
  box-sizing: border-box; /* Include padding and border in the element's width */
}

.submit-button {
  background-color: #4CAF50;
  color: white;
  padding: 10px 15px;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  width: 100%; /* Make the button full-width */
}

.submit-button:hover {
  background-color: #45a049;
}

.error {
  color: red;
  margin-top: 10px;
}
</style>
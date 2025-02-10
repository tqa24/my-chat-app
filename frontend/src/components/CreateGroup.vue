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

export default {
  name: 'CreateGroup',
  setup() {
    const groupName = ref('');
    const error = ref('');
    const router = useRouter();

    const createGroup = async () => {
      try {
        const response = await axios.post('http://localhost:8080/groups', {
          name: groupName.value,
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
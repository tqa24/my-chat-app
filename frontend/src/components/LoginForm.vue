<template>
  <div class="form-container">
    <h2>Login</h2>
    <form @submit.prevent="handleSubmit">
      <div class="form-group">
        <label for="identifier">Username or Email:</label>
        <input type="text" id="identifier" v-model.trim="identifier" required>
      </div>
      <div class="form-group">
        <label for="password">Password:</label>
        <input type="password" id="password" v-model.trim="password" required>
      </div>
      <button type="submit" class="submit-button">Login</button>
      <p v-if="error" class="error">{{ error }}</p>
    </form>
  </div>
</template>

<script>
import axios from 'axios';
import { useRouter } from 'vue-router';
import { useStore } from 'vuex';
import { ref } from 'vue';

export default {
  setup() {
    const identifier = ref('');
    const password = ref('');
    const error = ref('');
    const router = useRouter();
    const store = useStore();

    const instance = axios.create({
      baseURL: '/api', // Set base URL for all axios requests
    });

    const handleSubmit = async () => {
      try {
        const response = await instance.post('/login', {
          identifier: identifier.value,
          password: password.value,
        });
        // Store the user and token in Vuex
        store.dispatch('login', response.data.user);
        router.push('/'); // Redirect to home page
      } catch (err) {
        error.value = err.response?.data?.error || 'Login failed';
      }
    };

    return { identifier, password, error, handleSubmit };
  }
};
</script>

<style scoped>
.form-container {
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
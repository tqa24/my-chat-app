<template>
  <div class="form-container">
    <h2>Verify OTP</h2>
    <form @submit.prevent="verifyOTP">
      <div class="form-group">
        <label for="email">Email:</label>
        <input type="email" id="email" v-model="email" required>
      </div>
      <div class="form-group">
        <label for="otp">Enter OTP:</label>
        <input type="text" id="otp" v-model="otp" required>
      </div>
      <button type="submit">Verify OTP</button>
      <p v-if="error" class="error">{{ error }}</p>
    </form>
  </div>
</template>

<script>
import { useRouter } from 'vue-router';
import { ref } from 'vue';
import api from "@/store/api";

export default {
  setup() {
    const email = ref(''); // Email is now needed here
    const otp = ref('');
    const error = ref('');
    const router = useRouter();

    const instance = api;

    const verifyOTP = async () => {
      try {
        await instance.post('/verify-otp', {
          email: email.value, // Send email
          otp: otp.value,
        });
        router.push('/login'); // Redirect to login on success
      } catch (err) {
        error.value = err.response?.data?.error || 'OTP verification failed';
      }
    };

    return { email, otp, error, verifyOTP }; // Include email
  }
};
</script>

<style scoped>
.form-container {
  width: 300px;
  margin: 0 auto;
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

input[type="text"],
input[type="password"],
input[type="email"] {
  width: 100%;
  padding: 8px;
  border: 1px solid #ccc;
  border-radius: 4px;
  box-sizing: border-box;
}

.submit-button {
  background-color: #4CAF50;
  color: white;
  padding: 10px 15px;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  width: 100%;
}

.submit-button:hover {
  background-color: #45a049;
}

.error {
  color: red;
  margin-top: 10px;
}
</style>
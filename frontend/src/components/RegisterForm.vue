<template>
  <div class="form-container">
    <h2 v-if="!showOTPForm">Register</h2>
    <h2 v-else>Verify Email</h2>
    <form @submit.prevent="handleSubmit" v-if="!showOTPForm">
      <div class="form-group">
        <label for="username">Username:</label>
        <input type="text" id="username" v-model="username" required>
      </div>
      <div class="form-group">
        <label for="email">Email:</label>
        <input type="email" id="email" v-model="email" required>
        <p v-if="emailError" class="error">{{ emailError }}</p>
      </div>
      <div class="form-group">
        <label for="password">Password:</label>
        <input type="password" id="password" v-model="password" required>
      </div>
      <!-- Disable the button if the form is invalid -->
      <button type="submit" class="submit-button" :disabled="!isFormValid">Register</button>
      <p v-if="error" class="error">{{ error }}</p>
      <p class="login-link">
        Already have an account? <router-link to="/login">>> Login</router-link>
      </p>

    </form>

    <!-- OTP Form -->
    <form @submit.prevent="verifyOTP" v-else>
      <div class="form-group">
        <label for="otp">Enter OTP:</label>
        <input type="text" id="otp" v-model="otp" required>
      </div>
      <button type="submit">Verify OTP</button>
      <p v-if="otpError" class="error">{{ otpError }}</p>
    </form>
  </div>
</template>

<script>
import { useRouter } from 'vue-router';
import { ref, computed } from 'vue';
import api from "@/store/api";
export default {
  setup() {
    const username = ref('');
    const email = ref('');
    const password = ref('');
    const error = ref('');
    const router = useRouter();
    const showOTPForm = ref(false);
    const otp = ref('');
    const otpError = ref('');
    const emailError = ref('');

    const instance = api;

    const handleSubmit = async () => {
      emailError.value = ""; // Reset email error
      // Validate email format
      if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.value)) {
        emailError.value = "Invalid email format."; // Set error
        return; // Stop if the format is invalid
      }
      try {
        await instance.post('/register', {
          username: username.value,
          email: email.value,
          password: password.value,
        });
        // Show OTP form after successful registration attempt
        showOTPForm.value = true;
        error.value = ""; // Clear any previous errors
      } catch (err) {
        error.value = err.response?.data?.error || 'Registration failed';
      }
    };

    const verifyOTP = async () => {
      try {
        await instance.post('/verify-otp', {
          email: email.value,
          otp: otp.value,
        });
        // If OTP verification is successful, redirect to login.
        router.push('/login');
      } catch (err) {
        otpError.value = err.response?.data?.error || 'OTP verification failed';
      }
    };

    // Computed property for form validity
    const isFormValid = computed(() => {
      return username.value && /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.value) && password.value;
    });

    return {
      username,
      email,
      password,
      error,
      handleSubmit,
      showOTPForm,
      otp,
      verifyOTP,
      otpError,
      isFormValid,  // Return isFormValid
      emailError
    };
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
.login-link {
  text-align: center;
  margin-top: 15px;
  font-size: 0.9em;
}
</style>
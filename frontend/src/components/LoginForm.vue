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
      <p class="register-link">
        Don't have an account? <router-link to="/register">>> Register</router-link>
      </p>
      <!-- Add Resend OTP Link -->
      <p v-if="showResendOTP" class="resend-otp">
        Didn't receive the OTP? <a href="#" @click.prevent="resendOTP">Resend OTP</a>
      </p>
      <!-- Add Already Have OTP Link -->
      <p v-if="showResendOTP" class="already-have-otp">
        Already have an OTP? <router-link to="/verify-otp">Verify OTP</router-link>
      </p>
      <p v-if="resendMessage" class="resend-message">{{ resendMessage }}</p>
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
    const showResendOTP = ref(false); // Add state for showing the resend link
    const resendMessage = ref('');


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
        // Show resend OTP link if the error is "account not verified"
        if (err.response?.data?.error === "account not verified. Please check your email for the OTP") {
          showResendOTP.value = true;
        }
        error.value = err.response?.data?.error || 'Login failed';
      }
    };
    const resendOTP = async () => {
      try {
        // We only need email to resend otp
        let email = "";
        if(identifier.value.includes("@")){
          email = identifier.value
        } else {
          //Make email from username
          const user = await instance.get(`/profile?userID=${identifier.value}`)
          email = user.data.email
        }
        // Make API call to /api/resend-otp
        await instance.post('/resend-otp', { email: email });
        // Redirect to the OTP verification page on success
        router.push('/verify-otp');

      } catch (err) {
        // Show appropriate error
        resendMessage.value = err.response?.data?.error || 'Failed to resend OTP.';
      }
    };

    return { identifier, password, error, handleSubmit, showResendOTP, resendOTP, resendMessage }; // Return showResendOTP and resendOTP
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
.register-link {
  text-align: center;
  margin-top: 15px;
  font-size: 0.9em;
}
.resend-otp a {
  color: #007bff; /* Or any color you prefer */
  text-decoration: underline;
  cursor: pointer;
}
.resend-message{
  color: green;
}
.already-have-otp a {
  color: #007bff;
  text-decoration: underline;
}
</style>
import axios from 'axios';
import router from '../router/index.js';
import store from './index.js';

// Create a base URL that works in both development and production
const baseURL = process.env.NODE_ENV === 'development'
    ? 'http://localhost:8080/api'  // Development API URL
    : '/api';                      // Production API URL (relative path)

const api = axios.create({
    baseURL: baseURL,
});

// Add a request interceptor
api.interceptors.request.use(
    config => {
        const token = localStorage.getItem('token');
        if (token) {
            config.headers.Authorization = `Bearer ${token}`;
        }
        return config;
    },
    error => {
        return Promise.reject(error);
    }
);

api.interceptors.response.use(
    response => response,
    error => {
        if (error.response && error.response.status === 401) {
            // Token expired or invalid
            store.dispatch('logout');
            router.push('/login');
        }
        return Promise.reject(error);
    }
);

export default api;

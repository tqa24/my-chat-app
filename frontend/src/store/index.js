import { createStore } from 'vuex';
import axios from 'axios';
export default createStore({
    state: {
        user: null, // Store user information
        messages: [], // Store chat messages
        usersOnline: [], //Online user
        typingUsers: [], // Type user
        ws: null, // Add WebSocket
        selectedGroup: null,
        unreadCounts: {}, // Store unread counts for users and groups
    },
    mutations: {
        setUser(state, user) {
            state.user = user;
            if(user){
                localStorage.setItem('user', JSON.stringify(user)); // Persist user
            } else {
                localStorage.removeItem('user')
            }
        },
        setMessages(state, messages) {
            state.messages = messages;
        },
        addMessage(state, message) {
            // Add new message to the end (it will appear at the bottom due to flex-direction: column-reverse)
            state.messages.push(message);
        },
        addMessages(state, newMessages) {
            // Add new messages while maintaining order (newest first)
            const uniqueMessages = newMessages.filter(
                newMsg => !state.messages.some(
                    existingMsg => existingMsg.id === newMsg.id
                )
            );
            state.messages = [...uniqueMessages, ...state.messages];
        },
        setUsersOnline(state, users) {
            state.usersOnline = users;
        },
        addTypingUser(state, username) {
            if (!state.typingUsers.includes(username)) {
                state.typingUsers.push(username);
            }
        },
        removeTypingUser(state, username) {
            state.typingUsers = state.typingUsers.filter(u => u !== username);
        },
        clearMessages(state) {
            state.messages = [];
        },
        setWs(state, wsInstance) {
            state.ws = wsInstance;
        },
        setSelectedGroup(state, group) {
            state.selectedGroup = group;
        },
        // --- New Mutations for Unread Counts ---
        setUnreadCount(state, { id, count }) {
            state.unreadCounts = { ...state.unreadCounts, [id]: count };
            localStorage.setItem('unreadCounts', JSON.stringify(state.unreadCounts));
        },
        incrementUnreadCount(state, id) {
            const stringId = id?.toString();
            if (!stringId) {
                console.warn('Attempted to increment unread count with invalid ID:', id);
                return;
            }

            console.log('Incrementing unread count:', {
                id: stringId,
                currentCount: state.unreadCounts[stringId] || 0,
                newCount: (state.unreadCounts[stringId] || 0) + 1
            });

            state.unreadCounts = {
                ...state.unreadCounts,
                [stringId]: (state.unreadCounts[stringId] || 0) + 1
            };

            localStorage.setItem('unreadCounts', JSON.stringify(state.unreadCounts));
        },
        clearUnreadCount(state, id) {
            const stringId = id?.toString();
            if (!stringId) {
                console.warn('Attempted to clear unread count with invalid ID:', id);
                return;
            }

            console.log('Clearing unread count:', {
                id: stringId,
                previousCount: state.unreadCounts[stringId] || 0
            });

            state.unreadCounts = {
                ...state.unreadCounts,
                [stringId]: 0
            };

            localStorage.setItem('unreadCounts', JSON.stringify(state.unreadCounts));
        },
        // initialize unread counts from localStorage
        initializeUnreadCounts(state) {
            const saved = localStorage.getItem('unreadCounts');
            if (saved) {
                state.unreadCounts = JSON.parse(saved);
            }
        },
    },
    actions: {
        login({ commit }, user) {
            commit('setUser', user);
            // Set the Authorization header after successful login
            const token = localStorage.getItem('token');
            if (token) {
                axios.defaults.headers.common['Authorization'] = `Bearer ${token}`;
            }
        },
        logout({ commit }) {
            // Clear the Authorization header on logout
            delete axios.defaults.headers.common['Authorization'];
            commit('setUser', null);
            localStorage.removeItem('token'); // Remove the token on logout
            //Close websocket
            if (this.state.ws) {
                this.state.ws.close();
            }
        },
        setMessages({ commit }, messages) {
            commit('setMessages', messages);
        },
        addMessage({commit}, message){
            commit('addMessage', message)
        },
        setUsersOnline({ commit }, users) {
            commit('setUsersOnline', users);
        },
        addTypingUser({ commit }, username) {
            commit('addTypingUser', username);
        },
        removeTypingUser({ commit }, username) {
            commit('removeTypingUser', username);
        },
        clearMessages({commit}){
            commit('clearMessages')
        },
        setWs({ commit }, wsInstance) {
            commit('setWs', wsInstance);
        },
        setSelectedGroup({ commit }, group) {
            commit('setSelectedGroup', group);
        },
        // --- New Actions for Unread Counts ---
        incrementUnreadCount({ commit }, id) {
            commit('incrementUnreadCount', id);
        },

        markAsRead({ commit }, id) {
            commit('clearUnreadCount', id);
        },

        initializeUnreadCounts({ commit }) {
            commit('initializeUnreadCounts');
        }
    },
    getters: {
        currentUser: state => state.user,
        allMessages: state => state.messages,
        getUsersOnline: state => state.usersOnline,
        typingUsers: state => state.typingUsers,
        getUnreadCount: (state) => (id) => {
            const stringId = id?.toString();
            const count = stringId ? (state.unreadCounts[stringId] || 0) : 0;
            console.log(`Getting unread count for ${stringId}:`, count);
            return count;
        }
    },
});
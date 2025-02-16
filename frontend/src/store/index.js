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
        replyingTo: null, // NEW: Message being replied to
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
            state.messages.push(message);

            if (message.reply_to_message_id) {
                const originalMessageIndex = state.messages.findIndex(m => m.id === message.reply_to_message_id);
                if (originalMessageIndex !== -1) {
                    // Directly modify properties of the existing object.  Vue3's reactivity
                    // system will detect changes to nested properties.
                    state.messages[originalMessageIndex].reply_to_message = {
                        id: message.id, // The ID of the *reply* message
                        content: message.content,
                        sender_id: message.sender_id,
                        // Add other relevant fields from 'message' if needed
                    };
                }
            }
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
        setReplyingTo(state, message) { // NEW: Set the replyingTo message
            state.replyingTo = message;
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
        },
        // Refetch group message
        async fetchGroupMessages({dispatch, state}){
            if(state.selectedGroup && state.selectedGroup.id){
                try {
                    const response = await axios.get(
                        `http://localhost:8080/groups/${state.selectedGroup.id}/messages?page=1&pageSize=20` // Hard code for now
                    );
                    // Replace message with new message from server
                    dispatch('setMessages', response.data.messages.reverse());

                } catch (error) {
                    console.error("Failed to fetch group messages:", error);
                }
            }
        },
        // Refetch user message
        async fetchMessages({dispatch, state}){
            if(state.selectedUser){
                try {
                    const response = await axios.get(
                        //Change here
                        `http://localhost:8080/messages?user1=${state.user?.id}&user2=${state.selectedUser?.id}&page=1&pageSize=20` //Hard code
                    );
                    // Replace message with new message from server
                    dispatch('setMessages', response.data.messages.reverse());
                } catch (error) {
                    console.error("Failed to fetch messages:", error);
                }
            }
        },
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
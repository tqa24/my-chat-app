<template>
  <div class="home">
    <router-link to="/create-group">Create Group</router-link>
    <ChatWindow />
  </div>
</template>

<script>
import ChatWindow from '../components/ChatWindow.vue';
export default {
  components: {
    ChatWindow
  }
}
</script>

<style scoped>
.home a{
  display: block;
  margin-bottom: 20px;
}
</style>
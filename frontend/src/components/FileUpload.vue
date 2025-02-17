<template>
  <div class="file-upload">
    <input type="file" ref="fileInput" @change="handleFileChange" style="display: none;" />
    <button type="button" @click="triggerFileSelect">
      <i class="fas fa-paperclip"></i> Attach File <!-- FontAwesome icon -->
    </button>
    <span v-if="selectedFile" class="file-info">
        {{ selectedFile.name }} ({{ formatFileSize(selectedFile.size) }})
         <button type="button" @click="removeFile" class="remove-file">
            <i class="fas fa-times"></i>
        </button>
    </span>
    <span v-if="uploadError" class="upload-error">{{ uploadError }}</span>
  </div>
</template>

<script>
import axios from 'axios';
import { ref } from 'vue';

export default {
  emits: ['file-uploaded', 'file-removed'], // Emit events
  setup(props, { emit }) { // Use emit from context
    const fileInput = ref(null);
    const selectedFile = ref(null);
    const uploadError = ref('');

    const triggerFileSelect = () => {
      fileInput.value.click();
    };

    const handleFileChange = (event) => {
      const file = event.target.files[0];
      if (file) {
        if (file.size > 25 * 1024 * 1024) { // 25MB limit
          uploadError.value = 'File is too large (max 25MB)';
          selectedFile.value = null;
          emit('file-removed');
          return;
        }
        selectedFile.value = file;
        uploadError.value = ''; // Clear any previous error
        uploadFile(); // Automatically upload
      } else {
        selectedFile.value = null;
        emit('file-removed');
      }
    };

    const uploadFile = async () => {
      if (!selectedFile.value) return;

      const formData = new FormData();
      formData.append('file', selectedFile.value);

      try {
        const response = await axios.post('http://localhost:8080/upload', formData, {
          headers: {
            'Content-Type': 'multipart/form-data', // Important for file uploads
          },
        });
        // Emit the file information on successful upload
        emit('file-uploaded', {
          name: response.data.filename, // Use the unique filename from server
          path: response.data.filepath,
          type: response.data.filetype,
          size: response.data.filesize,
        });
        uploadError.value = '';
      } catch (error) {
        console.error("Upload error:", error.response || error); // Log full error
        uploadError.value = 'Upload failed: ' + (error.response ? error.response.data.error : error.message);
        emit('file-removed')
        selectedFile.value = null; // Clear on error
      }
    };
    const removeFile = () => {
      selectedFile.value = null; // Clear the selected file
      fileInput.value.value = null; // Reset the file input (important!)
      uploadError.value = '';      // Clear any error message
      emit('file-removed');  // Emit event when file is removed
    };
    // Helper function to format file sizes
    const formatFileSize = (bytes) => {
      if (bytes === 0) return '0 Bytes';
      const k = 1024;
      const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
      const i = Math.floor(Math.log(bytes) / Math.log(k));
      return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    };


    return { fileInput, selectedFile, uploadError, triggerFileSelect, handleFileChange, removeFile, formatFileSize };
  },
};
</script>

<style scoped>
.file-upload {
  display: flex;
  align-items: center;
  margin-bottom: 5px; /* Add some spacing */
}

.file-info {
  margin-left: 8px;
  margin-right: 8px;
  font-size: 0.9em;
}
.upload-error {
  color: red;
  margin-left: 8px;
  font-size: 0.9em;
}
.remove-file{
  background: none;
  border: none;
  padding: 0;
}
</style>
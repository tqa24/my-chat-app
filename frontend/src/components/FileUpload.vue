<template>
  <div class="file-upload">
    <input type="file" ref="fileInput" @change="handleFileChange" style="display: none;" />
    <button type="button" @click="triggerFileSelect">
      <i class="fas fa-paperclip"></i> Attach File
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
import { sha256 } from 'js-sha256'; // Import a SHA-256 library

export default {
  emits: ['file-uploaded', 'file-removed'],
  setup(props, { emit }) {
    const fileInput = ref(null);
    const selectedFile = ref(null);
    const uploadError = ref('');

    const triggerFileSelect = () => {
      fileInput.value.click();
    };

    const handleFileChange = async (event) => {
      const file = event.target.files[0];
      if (file) {
        if (file.size > 25 * 1024 * 1024) {
          uploadError.value = 'File is too large (max 25MB)';
          selectedFile.value = null;
          emit('file-removed');
          return;
        }

        // Calculate SHA-256 checksum *before* upload.
        const arrayBuffer = await file.arrayBuffer();
        const hash = sha256(arrayBuffer);

        selectedFile.value = file;
        uploadError.value = '';
        uploadFile(hash); // Pass the checksum
      } else {
        selectedFile.value = null;
        emit('file-removed');
      }
    };

    const uploadFile = async (checksum) => { // Add checksum parameter
      if (!selectedFile.value) return;

      const formData = new FormData();
      formData.append('file', selectedFile.value);
      formData.append('checksum', checksum); // Send checksum with the request

      try {
        const response = await axios.post('http://localhost:8080/upload', formData, {
          headers: {
            'Content-Type': 'multipart/form-data',
          },
        });

        if (response.data.duplicate) {
          // Handle duplicate file.  Don't treat as an error.
          emit('file-uploaded', {
            name: response.data.filename,
            path: response.data.filepath,
            type: response.data.filetype,
            size: response.data.filesize,
            checksum: response.data.checksum, // Include for consistency
          });
          uploadError.value = ''; // Clear any error
        } else {
          // Handle new file upload.
          emit('file-uploaded', {
            name: response.data.filename,
            path: response.data.filepath,
            type: response.data.filetype,
            size: response.data.filesize,
            checksum: response.data.checksum,
          });
          uploadError.value = '';
        }
      } catch (error) {
        console.error("Upload error:", error.response || error);
        uploadError.value = 'Upload failed: ' + (error.response ? error.response.data.error : error.message);
        emit('file-removed');
        selectedFile.value = null;
      }
    };

    const removeFile = () => {
      selectedFile.value = null;
      fileInput.value.value = null;
      uploadError.value = '';
      emit('file-removed');
    };

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
<template>
  <div class="file-upload">
    <input type="file" ref="fileInput" @change="handleFileChange" style="display: none;" accept="*" />
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

    <!-- Image Preview -->
    <div v-if="previewUrl && isImage" class="image-preview-container">
      <img :src="previewUrl" alt="Image Preview" class="image-preview" />
    </div>
  </div>
</template>

<script>
import axios from 'axios';
import { ref } from 'vue';
import { sha256 } from 'js-sha256';

export default {
  emits: ['file-uploaded', 'file-removed'],
  setup(props, { emit }) {
    const fileInput = ref(null);
    const selectedFile = ref(null);
    const uploadError = ref('');
    const previewUrl = ref(''); // Store the preview URL
    const isImage = ref(false); // Track if the selected file is an image

    const triggerFileSelect = () => {
      fileInput.value.click();
    };

    const handleFileChange = async (event) => {
      const file = event.target.files[0];
      if (file) {
        if (file.size > 25 * 1024 * 1024) {
          uploadError.value = 'File is too large (max 25MB)';
          selectedFile.value = null;
          previewUrl.value = ''; // Clear preview
          isImage.value = false;
          emit('file-removed');
          return;
        }

        selectedFile.value = file;
        uploadError.value = '';
        isImage.value = file.type.startsWith('image/'); // Check if it's an image

        // *** Generate Preview URL *** (only if it's an image)
        if (isImage.value) {
          const reader = new FileReader();
          reader.onload = (e) => {
            previewUrl.value = e.target.result;
          };
          reader.readAsDataURL(file);
        } else {
          previewUrl.value = ''; // Clear preview if not an image
        }

        // Calculate SHA-256 checksum *before* upload.
        const arrayBuffer = await file.arrayBuffer();
        const hash = sha256(arrayBuffer);
        uploadFile(hash); // Pass the checksum
      } else {
        selectedFile.value = null;
        previewUrl.value = ''; // Clear preview on file removal
        isImage.value = false;
        emit('file-removed');
      }
    };

    const uploadFile = async (checksum) => {
      if (!selectedFile.value) return;

      const formData = new FormData();
      formData.append('file', selectedFile.value);
      formData.append('checksum', checksum);

      try {
        const response = await axios.post('http://localhost:8080/upload', formData, {
          headers: {
            'Content-Type': 'multipart/form-data',
          },
        });

        if (response.data.duplicate) {
          emit('file-uploaded', {
            name: response.data.filename,
            path: response.data.filepath,
            type: response.data.filetype,
            size: response.data.filesize,
            checksum: response.data.checksum,
          });
          uploadError.value = '';
        } else {
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
        previewUrl.value = ''; // Clear preview on error
        isImage.value = false;
      }
    };

    const removeFile = () => {
      selectedFile.value = null;
      fileInput.value.value = null;
      uploadError.value = '';
      previewUrl.value = ''; // Clear the preview URL
      isImage.value = false;
      emit('file-removed');
    };

    const formatFileSize = (bytes) => {
      if (bytes === 0) return '0 Bytes';
      const k = 1024;
      const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
      const i = Math.floor(Math.log(bytes) / Math.log(k));
      return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    };

    return { fileInput, selectedFile, uploadError, previewUrl, isImage, triggerFileSelect, handleFileChange, removeFile, formatFileSize };
  },
};
</script>

<style scoped>
.file-upload {
  display: flex;
  align-items: center;
  margin-bottom: 5px;
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

/* Style for the image preview */
.image-preview-container {
  margin-top: 5px;
  max-width: 100%; /* Prevent overflow */
}

.image-preview {
  max-width: 100px; /* Control preview size */
  max-height: 100px;
  border: 1px solid #ccc;
  border-radius: 4px;
}
</style>
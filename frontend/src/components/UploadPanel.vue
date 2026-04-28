<template>
  <div style="display:flex;align-items:center;gap:8px">
    <el-date-picker v-model="date" type="date" value-format="YYYY-MM-DD"
      placeholder="默认今日" style="width:150px" size="default" />
    <el-upload :before-upload="handleUpload" accept=".csv" :show-file-list="false">
      <el-button type="primary" :loading="loading">
        <el-icon><Upload /></el-icon>&nbsp;上传CSV
      </el-button>
    </el-upload>
    <el-tooltip content="表头须含：用户名、武勋、繁荣（逗号分隔）" placement="bottom">
      <el-icon style="color:#909399;cursor:pointer"><QuestionFilled /></el-icon>
    </el-tooltip>
    <el-alert v-if="result" :title="result.message" :type="result.type"
      :description="result.detail" show-icon closable style="max-width:320px" />
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { uploadExcel } from '../api'

const emit = defineEmits(['uploaded'])
const date = ref('')
const loading = ref(false)
const result = ref(null)

async function handleUpload(file) {
  loading.value = true
  result.value = null
  try {
    const { data } = await uploadExcel(file, date.value)
    result.value = {
      type: 'success',
      message: `上传成功：导入 ${data.imported} 条，跳过 ${data.skipped} 条（${data.date}）`
    }
    emit('uploaded')
  } catch (e) {
    result.value = {
      type: 'error',
      message: e.response?.data?.error || '上传失败'
    }
  } finally {
    loading.value = false
  }
  return false
}
</script>

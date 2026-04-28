<template>
  <div class="page">
    <div style="margin-bottom:20px;display:flex;align-items:center;gap:16px">
      <span style="font-size:18px;font-weight:600">云游联盟管理系统</span>
      <UploadPanel @uploaded="onUploaded" />
    </div>

    <el-tabs v-model="tab">
      <el-tab-pane label="全盟数据" name="table">
        <MemberTable ref="tableRef" />
      </el-tab-pane>
      <el-tab-pane label="变化曲线" name="chart" lazy>
        <MemberChart ref="chartRef" />
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import UploadPanel from './components/UploadPanel.vue'
import MemberChart from './components/MemberChart.vue'
import MemberTable from './components/MemberTable.vue'

const tab = ref('table')
const tableRef = ref(null)
const chartRef = ref(null)

function onUploaded() {
  tableRef.value?.refresh()
  chartRef.value?.refresh()
}
</script>

<style>
body { margin: 0; background: #f5f7fa; }
.page { max-width: 1400px; margin: 0 auto; padding: 20px 24px; }
</style>

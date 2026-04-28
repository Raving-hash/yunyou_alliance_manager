<template>
  <div>
    <!-- 联盟总武勋 -->
    <el-card style="margin-bottom:16px">
      <template #header><span>联盟武勋变化</span></template>
      <div v-show="!allianceReady" style="text-align:center;color:#999;padding:24px">上传数据后显示</div>
      <div ref="allianceChartEl" style="height:260px" />
    </el-card>

    <!-- 个人图表 -->
    <el-card>
      <template #header>
        <div style="display:flex;align-items:center;gap:12px">
          <span>成员详情</span>
          <el-select v-model="selectedUsername" placeholder="选择成员" clearable
            style="width:160px" @change="loadMember">
            <el-option v-for="m in members" :key="m.username" :label="m.username" :value="m.username" />
          </el-select>
        </div>
      </template>

      <div v-if="!memberData" style="text-align:center;color:#999;padding:24px">请选择成员</div>
      <template v-else>
        <el-row :gutter="12">
          <el-col :span="12">
            <div ref="prosperityChartEl" style="height:240px" />
          </el-col>
          <el-col :span="12">
            <div ref="militaryChartEl" style="height:240px" />
          </el-col>
        </el-row>
        <div ref="rateChartEl" style="height:240px;margin-top:12px" />
      </template>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted, nextTick } from 'vue'
import * as echarts from 'echarts'
import { getMembers, getChartByMember, getAllianceChart } from '../api'

const members = ref([])
const selectedUsername = ref(null)
const allianceReady = ref(false)
const memberData = ref(null)

const allianceChartEl = ref(null)
const prosperityChartEl = ref(null)
const militaryChartEl = ref(null)
const rateChartEl = ref(null)

let allianceInstance = null
let prosperityInstance = null
let militaryInstance = null
let rateInstance = null

onMounted(async () => {
  // div 已在 DOM 中，立即初始化 ECharts 实例
  allianceInstance = echarts.init(allianceChartEl.value)
  await fetchMembers()
  await loadAlliance()
})

async function fetchMembers() {
  const { data } = await getMembers()
  members.value = data || []
}

async function loadAlliance() {
  try {
    const { data } = await getAllianceChart()
    if (!data.dates?.length) return
    allianceReady.value = true
    allianceInstance.setOption(buildAllianceOption(data))
  } catch (e) {
    console.error('loadAlliance error:', e)
  }
}

async function loadMember() {
  if (!selectedUsername.value) { memberData.value = null; return }
  const { data } = await getChartByMember(selectedUsername.value)
  memberData.value = data
  await nextTick()
  if (!prosperityInstance) prosperityInstance = echarts.init(prosperityChartEl.value)
  if (!militaryInstance)   militaryInstance   = echarts.init(militaryChartEl.value)
  if (!rateInstance)       rateInstance       = echarts.init(rateChartEl.value)
  prosperityInstance.setOption(buildLine(`${data.member.username} - 繁荣`, data.dates, data.prosperity, '#91cc75', ''))
  militaryInstance.setOption(buildLine(`${data.member.username} - 武勋`, data.dates, data.military.map(toWan), '#5470c6', '万'))
  rateInstance.setOption(buildRateOption(data))
}

defineExpose({
  refresh: async () => {
    await fetchMembers()
    await loadAlliance()
    if (selectedUsername.value) await loadMember()
  }
})

// --- option builders ---

function toWan(v) { return v == null ? null : +(v / 10000).toFixed(2) }

function buildAllianceOption(data) {
  return {
    title: { text: '联盟每日武勋总量', left: 'center', textStyle: { fontSize: 13 } },
    tooltip: { trigger: 'axis' },
    legend: { top: 24, data: ['武勋总量', '每日增量'] },
    xAxis: { type: 'category', data: data.dates, boundaryGap: true },
    yAxis: [
      { type: 'value', name: '万', position: 'left' },
      { type: 'value', name: '万', position: 'right' }
    ],
    series: [
      { name: '武勋总量', type: 'line', smooth: true, yAxisIndex: 0, data: data.total_military.map(toWan), itemStyle: { color: '#5470c6' } },
      { name: '每日增量', type: 'bar', yAxisIndex: 1, data: data.military_delta.map(toWan), itemStyle: { color: '#fac858' } }
    ]
  }
}

function buildLine(title, dates, values, color, unit) {
  return {
    title: { text: title, left: 'center', textStyle: { fontSize: 13 } },
    tooltip: {
      trigger: 'axis',
      valueFormatter: v => v == null ? '-' : v + unit
    },
    xAxis: { type: 'category', data: dates, boundaryGap: false },
    yAxis: { type: 'value' },
    series: [{ type: 'line', smooth: true, data: values, itemStyle: { color } }]
  }
}

function buildRateOption(data) {
  const ratePercent = data.military_rate.map(v => v == null ? null : +(v * 100).toFixed(4))
  return {
    title: { text: `${data.member.username} - 武勋率（当日武勋增量/繁荣）`, left: 'center', textStyle: { fontSize: 13 } },
    tooltip: {
      trigger: 'axis',
      valueFormatter: v => v == null ? '-' : v + '%'
    },
    xAxis: { type: 'category', data: data.dates, boundaryGap: false },
    yAxis: { type: 'value', axisLabel: { formatter: v => v + '%' } },
    series: [{
      type: 'line', smooth: true,
      data: ratePercent,
      connectNulls: false,
      itemStyle: { color: '#ee6666' }
    }]
  }
}
</script>

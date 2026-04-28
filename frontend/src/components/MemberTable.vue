<template>
  <el-card>
    <template #header>
      <div style="display:flex;align-items:center;gap:12px;flex-wrap:wrap">
        <span>全盟数据</span>
        <el-input
          v-model="search"
          placeholder="按名字搜索…"
          clearable
          style="width:200px"
          @input="onSearch"
        >
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-text type="info" size="small">共 {{ list.length }} 人</el-text>
        <el-tag v-if="allianceAvg != null" type="warning" size="default">
          全盟平均武勋增长率：{{ fmtRate(allianceAvg) }}
        </el-tag>
      </div>
    </template>

    <el-table :data="sortedList" stripe border v-loading="loading" empty-text="暂无数据">
      <el-table-column label="名字" width="120" fixed>
        <template #default="{ row }">
          <span :style="nameColor(row.GrowthRateDiff)">{{ row.Username }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="CurrentMilitary" label="当前武勋" width="110" sortable
        :sort-method="(a,b) => a.CurrentMilitary - b.CurrentMilitary" />
      <el-table-column prop="CurrentProsperity" label="当前繁荣" width="110" sortable
        :sort-method="(a,b) => a.CurrentProsperity - b.CurrentProsperity" />

      <!-- 武勋增长率 & 与均值差 -->
      <el-table-column label="武勋增长率" width="110" sortable
        :sort-method="(a,b) => (a.GrowthRate ?? -Infinity) - (b.GrowthRate ?? -Infinity)">
        <template #default="{ row }">
          <span v-if="row.GrowthRate != null" :style="rateColor(row.GrowthRate)">
            {{ fmtRate(row.GrowthRate) }}
          </span>
          <span v-else style="color:#ccc">-</span>
        </template>
      </el-table-column>

      <el-table-column label="与均值差" width="110" sortable
        :sort-method="(a,b) => (a.GrowthRateDiff ?? -Infinity) - (b.GrowthRateDiff ?? -Infinity)">
        <template #default="{ row }">
          <span v-if="row.GrowthRateDiff != null" :style="rateColor(row.GrowthRateDiff)">
            {{ row.GrowthRateDiff >= 0 ? '+' : '' }}{{ fmtRate(row.GrowthRateDiff) }}
          </span>
          <span v-else style="color:#ccc">-</span>
        </template>
      </el-table-column>

      <!-- 最近3日 -->
      <el-table-column
        v-for="(_, i) in 3" :key="i"
        :label="dayLabel(i)"
        min-width="210"
      >
        <template #default="{ row }">
          <template v-if="row.Recent && row.Recent[i]">
            <el-descriptions :column="2" size="small" border>
              <el-descriptions-item label="武勋">{{ row.Recent[i].Military }}</el-descriptions-item>
              <el-descriptions-item label="繁荣">{{ row.Recent[i].Prosperity }}</el-descriptions-item>
              <el-descriptions-item label="武勋增量" :span="2">
                <span v-if="row.Recent[i].MilitaryDelta != null"
                  :style="{ color: row.Recent[i].MilitaryDelta >= 0 ? '#67c23a' : '#f56c6c' }">
                  {{ row.Recent[i].MilitaryDelta >= 0 ? '+' : '' }}{{ row.Recent[i].MilitaryDelta }}
                </span>
                <span v-else style="color:#ccc">-</span>
              </el-descriptions-item>
            </el-descriptions>
          </template>
          <span v-else style="color:#ccc">-</span>
        </template>
      </el-table-column>
    </el-table>
  </el-card>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { getMembersOverview } from '../api'

const list = ref([])
const allianceAvg = ref(null)

// 红色（均值差 < -50%）置顶优先，其次绿色（> 50%），其余保持原序
const sortedList = computed(() => {
  const red    = list.value.filter(m => (m.GrowthRateDiff ?? 0) < -50)
  const green  = list.value.filter(m => (m.GrowthRateDiff ?? 0) > 50)
  const normal = list.value.filter(m => {
    const d = m.GrowthRateDiff ?? 0
    return d >= -50 && d <= 50
  })
  return [...green, ...red, ...normal]
})

function nameColor(d) {
  if (d != null && d < -50) return { color: '#f56c6c', fontWeight: 'bold' }
  if (d != null && d > 50)  return { color: '#67c23a', fontWeight: 'bold' }
  return {}
}
const search = ref('')
const loading = ref(false)
let debounceTimer = null

onMounted(() => load())

async function load(q = '') {
  loading.value = true
  try {
    const { data } = await getMembersOverview(q)
    list.value = data.members || []
    allianceAvg.value = data.alliance_avg_growth_rate ?? null
  } finally {
    loading.value = false
  }
}

function onSearch() {
  clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => load(search.value.trim()), 300)
}

function dayLabel(i) {
  if (i === 0) return '最新一日'
  if (i === 1) return '前一日'
  return '前两日'
}

function fmtRate(v) {
  return (v >= 0 ? '' : '') + v.toFixed(2) + '%'
}

function rateColor(v) {
  return { color: v > 0 ? '#67c23a' : v < 0 ? '#f56c6c' : '#909399' }
}

defineExpose({ refresh: () => load(search.value.trim()) })
</script>


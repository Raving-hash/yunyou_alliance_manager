import axios from 'axios'

const http = axios.create({ baseURL: 'http://localhost:8080' })

export const uploadExcel = (file, date) => {
  const form = new FormData()
  form.append('file', file)
  if (date) form.append('date', date)
  return http.post('/api/upload', form)
}

export const getMembers = () => http.get('/api/members')

export const getChartByMember = (id) => http.get(`/api/chart/${id}`)

export const getAllianceChart = () => http.get('/api/chart/alliance')

export const getMembersOverview = (search = '') =>
  http.get('/api/members/overview', { params: search ? { search } : {} })

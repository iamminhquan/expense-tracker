import { ArcElement, BarElement, CategoryScale, Chart, LinearScale, Tooltip } from 'chart.js'

// Imported for its side effect: react-chartjs-2 throws at render without these.
Chart.register(ArcElement, BarElement, CategoryScale, LinearScale, Tooltip)

Chart.defaults.font.family = "'Be Vietnam Pro', system-ui, sans-serif"
Chart.defaults.font.size = 12

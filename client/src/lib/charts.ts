import { ArcElement, BarElement, CategoryScale, Chart, Legend, LinearScale, Tooltip } from 'chart.js'

// Registered once, imported for its side effect wherever a chart renders
// (react-chartjs-2's <Doughnut>/<Bar> throw at render time without this).
Chart.register(ArcElement, BarElement, CategoryScale, LinearScale, Tooltip, Legend)

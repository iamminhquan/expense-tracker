import { ArcElement, BarElement, CategoryScale, Chart, Legend, LinearScale, Tooltip } from 'chart.js'

// Imported for its side effect: react-chartjs-2 throws at render without these.
Chart.register(ArcElement, BarElement, CategoryScale, LinearScale, Tooltip, Legend)

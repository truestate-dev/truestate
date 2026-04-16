// TrendChart — pure SVG trend chart for evaluation history.
// No chart library dependency; works with the existing React + TS setup.

import { useNavigate } from 'react-router-dom'
import { EvaluationSummary } from '../api/client'

interface Props {
  data: EvaluationSummary[]
}

// Layout constants (SVG coordinate space)
const W = 560
const H = 110
const PAD = { top: 12, right: 12, bottom: 28, left: 38 }
const PLOT_W = W - PAD.left - PAD.right
const PLOT_H = H - PAD.top - PAD.bottom

function buildPath(points: Array<[number, number]>): string {
  return points.map(([x, y], i) => `${i === 0 ? 'M' : 'L'} ${x.toFixed(1)} ${y.toFixed(1)}`).join(' ')
}

export default function TrendChart({ data }: Props) {
  const navigate = useNavigate()

  // Need at least 2 points for a line
  if (data.length < 2) {
    return (
      <p className="text-slate-300 text-xs text-center py-6">
        Run 2+ evaluations to see a trend.
      </p>
    )
  }

  // Sort oldest → newest
  const sorted = [...data].sort(
    (a, b) => new Date(a.evaluated_at).getTime() - new Date(b.evaluated_at).getTime()
  )

  const times = sorted.map((d) => new Date(d.evaluated_at).getTime())
  const tMin = times[0]
  const tMax = times[times.length - 1]
  const tRange = tMax - tMin || 1

  const maxFindings = Math.max(...sorted.map((d) => d.finding_count), 1)
  const maxDrift = Math.max(...sorted.map((d) => d.drift_count), 0)
  const yMax = Math.max(maxFindings, maxDrift, 1)

  const xOf = (t: number) => PAD.left + ((t - tMin) / tRange) * PLOT_W
  const yOf = (v: number) => PAD.top + (1 - v / yMax) * PLOT_H

  const findingsPts: Array<[number, number]> = sorted.map((d, i) => [xOf(times[i]), yOf(d.finding_count)])
  const driftPts: Array<[number, number]> = sorted.map((d, i) => [xOf(times[i]), yOf(d.drift_count)])

  // Y-axis labels (0, mid, max)
  const yTicks = [0, Math.round(yMax / 2), yMax]

  // X-axis date labels: first and last
  const fmtDate = (iso: string) =>
    new Date(iso).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })

  return (
    <div>
      {/* Legend */}
      <div className="flex gap-4 text-xs text-slate-500 mb-1">
        <span className="flex items-center gap-1.5">
          <span className="inline-block w-3 h-0.5 bg-red-400 rounded" />
          Findings
        </span>
        <span className="flex items-center gap-1.5">
          <span className="inline-block w-3 h-0.5 bg-amber-400 rounded" />
          Drift items
        </span>
      </div>

      <svg
        viewBox={`0 0 ${W} ${H}`}
        className="w-full"
        style={{ height: H }}
        aria-label="Evaluation trend"
      >
        {/* Y-axis grid lines + labels */}
        {yTicks.map((v) => {
          const y = yOf(v)
          return (
            <g key={v}>
              <line
                x1={PAD.left} y1={y} x2={W - PAD.right} y2={y}
                stroke="#e2e8f0" strokeWidth="1"
              />
              <text
                x={PAD.left - 4} y={y + 4}
                textAnchor="end" fontSize="9" fill="#94a3b8"
              >
                {v}
              </text>
            </g>
          )
        })}

        {/* X-axis */}
        <line
          x1={PAD.left} y1={PAD.top + PLOT_H}
          x2={W - PAD.right} y2={PAD.top + PLOT_H}
          stroke="#e2e8f0" strokeWidth="1"
        />

        {/* X-axis date labels: first, last, and one middle if ≥4 points */}
        {[0, sorted.length - 1, ...(sorted.length >= 4 ? [Math.floor(sorted.length / 2)] : [])].map((i) => (
          <text
            key={i}
            x={xOf(times[i])}
            y={H - 4}
            textAnchor="middle"
            fontSize="9"
            fill="#94a3b8"
          >
            {fmtDate(sorted[i].evaluated_at)}
          </text>
        ))}

        {/* Drift line */}
        {maxDrift > 0 && (
          <path
            d={buildPath(driftPts)}
            fill="none"
            stroke="#fbbf24"
            strokeWidth="1.5"
            strokeLinejoin="round"
          />
        )}

        {/* Findings line */}
        <path
          d={buildPath(findingsPts)}
          fill="none"
          stroke="#f87171"
          strokeWidth="2"
          strokeLinejoin="round"
        />

        {/* Dots — clickable, navigates to evaluation */}
        {sorted.map((d, i) => (
          <g key={d.id}>
            {/* Drift dot */}
            {maxDrift > 0 && (
              <circle
                cx={driftPts[i][0]} cy={driftPts[i][1]}
                r="3" fill="#fbbf24" stroke="white" strokeWidth="1"
                className="cursor-pointer"
                onClick={() => navigate(`/evaluations/${d.id}`)}
              >
                <title>{fmtDate(d.evaluated_at)}: {d.drift_count} drift</title>
              </circle>
            )}
            {/* Findings dot */}
            <circle
              cx={findingsPts[i][0]} cy={findingsPts[i][1]}
              r="3.5"
              fill={d.finding_count === 0 ? '#4ade80' : '#f87171'}
              stroke="white" strokeWidth="1.5"
              className="cursor-pointer"
              onClick={() => navigate(`/evaluations/${d.id}`)}
            >
              <title>{fmtDate(d.evaluated_at)}: {d.finding_count} findings</title>
            </circle>
          </g>
        ))}
      </svg>
    </div>
  )
}

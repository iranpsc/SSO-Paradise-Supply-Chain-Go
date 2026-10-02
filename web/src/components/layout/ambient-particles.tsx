import type { CSSProperties } from "react";

// Fixed positions keep server/client rendering identical. Only transform and
// opacity animate; this background has no timers, event listeners or hit targets.
export function AmbientParticles() {
  return (
    <div className="ambient-particles" aria-hidden="true">
      {Array.from({ length: 26 }, (_, index) => (
        <span
          key={index}
          style={
            {
              left: `${(index * 37 + 7) % 100}%`,
              top: `${(index * 23 + 11) % 100}%`,
              "--particle-size": `${3 + (index % 4)}px`,
              "--particle-duration": `${15 + (index % 9) * 3}s`,
              "--particle-delay": `${-index * 2}s`,
              "--particle-drift": `${index % 2 ? -35 : 35}px`,
            } as CSSProperties
          }
        />
      ))}
    </div>
  );
}

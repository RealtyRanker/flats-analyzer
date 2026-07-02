// Package geo computes a flat's position relative to Moscow's ring roads
// (Sadovoye Koltso, TTK).
package geo

import "math"

const earthRadiusMeters = 6371000.0

type point struct {
	lat, lon float64
}

// ring is a closed polygon of (lat, lon) vertices in traversal order (the
// last point implicitly connects back to the first), plus the local
// equirectangular projection used to measure distances to it.
type ring struct {
	points          []point
	referenceLatRad float64
}

// newRing builds a ring from its vertices, deriving a local projection
// reference latitude (the vertex mean) — accurate enough for a ring's
// city-scale extent, and far cheaper than great-circle math per edge.
func newRing(points []point) *ring {
	sum := 0.0
	for _, p := range points {
		sum += p.lat
	}
	return &ring{
		points:          points,
		referenceLatRad: (sum / float64(len(points))) * math.Pi / 180,
	}
}

// project converts (lat, lon) degrees to local planar meters around r's
// reference latitude.
func (r *ring) project(lat, lon float64) (x, y float64) {
	latRad := lat * math.Pi / 180
	lonRad := lon * math.Pi / 180
	x = earthRadiusMeters * lonRad * math.Cos(r.referenceLatRad)
	y = earthRadiusMeters * latRad
	return x, y
}

// inside reports whether (lat, lon) falls inside r, via the standard
// ray-casting (even-odd) algorithm.
func (r *ring) inside(lat, lon float64) bool {
	inside := false
	n := len(r.points)
	for i, j := 0, n-1; i < n; j, i = i, i+1 {
		pi, pj := r.points[i], r.points[j]
		if (pi.lat > lat) != (pj.lat > lat) {
			lonAtLat := pj.lon + (lat-pj.lat)*(pi.lon-pj.lon)/(pi.lat-pj.lat)
			if lon < lonAtLat {
				inside = !inside
			}
		}
	}
	return inside
}

// distanceKM returns the shortest distance in kilometers from (lat, lon) to
// r's boundary: the minimum, over every edge (segment between consecutive
// ring points, including the closing edge from the last point back to the
// first), of the point-to-segment distance.
func (r *ring) distanceKM(lat, lon float64) float64 {
	px, py := r.project(lat, lon)
	best := math.Inf(1)
	n := len(r.points)
	for i := 0; i < n; i++ {
		a := r.points[i]
		b := r.points[(i+1)%n]
		ax, ay := r.project(a.lat, a.lon)
		bx, by := r.project(b.lat, b.lon)
		if d := distancePointToSegment(px, py, ax, ay, bx, by); d < best {
			best = d
		}
	}
	return best / 1000
}

// distancePointToSegment returns the Euclidean distance from (px, py) to the
// closest point on segment [(ax, ay), (bx, by)].
func distancePointToSegment(px, py, ax, ay, bx, by float64) float64 {
	dx, dy := bx-ax, by-ay
	lengthSq := dx*dx + dy*dy
	if lengthSq == 0 {
		return math.Hypot(px-ax, py-ay)
	}
	t := ((px-ax)*dx + (py-ay)*dy) / lengthSq
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	closestX := ax + t*dx
	closestY := ay + t*dy
	return math.Hypot(px-closestX, py-closestY)
}

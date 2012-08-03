package dns

// A structure for handling zone data

import (
	"radix"
	"strings"
)

// Zone represents a DNS zone. 
type Zone struct {
	Origin       string // Origin of the zone
	*radix.Radix        // Zone data
}

// ZoneData holds all the RRs having their ownername equal to Name.
type ZoneData struct {
	Name string          // Domain name for this node
	RR   map[uint16][]RR // Map of the RR type to the RR
	// DNSSEC signatures for the RRsets
	Signatures []*RR_RRSIG
	// Always false, except for NSsets that differ from z.Origin
	NonAuth bool
}


// toRadixName reverses a domainname so that when we store it in the radix tree
// we preserve the nsec ordering of the zone (this idea was stolen from NSD).
// each label is also lowercased.
func toRadixName(d string) string {
	s := ""
	for _, l := range SplitLabels(d) {
		s = strings.ToLower(l) + "." + s
	}
	return s
}

// NewZone creates an initialized zone with Origin set to origin.
func NewZone(origin string) *Zone {
	if origin == "" {
		origin = "."
	}
	if _, _, ok := IsDomainName(origin); !ok {
		return nil
	}
	z := new(Zone)
	z.Origin = Fqdn(origin)
	z.Radix = radix.New()
	return z
}

// Insert inserts an RR into the zone. Duplicate data overwrites the old data.
func (z *Zone) Insert(r RR) error {
	if !IsSubDomain(r.Header().Name, z.Origin) {
		return &Error{Err: "out of zone data", Name: r.Header().Name}
	}

	key := toRadixName(r.Header().Name)
	zd := z.Radix.Find(key)
	if zd == nil {
		zd := new(ZoneData)
		zd.Name = r.Header().Name
		zd.RR = make(map[uint16][]RR)
		zd.Signatures = make([]*RR_RRSIG, 0)
		switch t := r.Header().Rrtype; t {
		case TypeRRSIG:
			zd.Signatures = append(zd.Signatures, r.(*RR_RRSIG))
		case TypeNS:
			// NS records with other names than z.Origin are non-auth
			if r.Header().Name != z.Origin {
				zd.NonAuth = true
			}
			fallthrough
		default:
			zd.RR[t] = append(zd.RR[t], r)
		}
		z.Radix.Insert(key, zd)
		return nil
	}
	// Name already there
	switch t := r.Header().Rrtype; t {
	case TypeRRSIG:
		zd.Value.(*ZoneData).Signatures = append(zd.Value.(*ZoneData).Signatures, r.(*RR_RRSIG))
	case TypeNS:
		if r.Header().Name != z.Origin {
			zd.Value.(*ZoneData).NonAuth = true
		}
		fallthrough
	default:
		zd.Value.(*ZoneData).RR[t] = append(zd.Value.(*ZoneData).RR[t], r)
	}
	return nil
}

// RemoveName removeRRset ??
func (z *Zone) Remove(r RR) error {
	return nil
}

// Find search the zone data and returns a node or nil when
// nothing is found.
func (z *Zone) Find(s string) *ZoneData {
	println("looking for", toRadixName(s))
	if z.Radix == nil {
		println("huh nil")
		return nil
	}
	zd := z.Radix.Find(toRadixName(s))
	if zd == nil {
		return nil
	}
	return zd.Value.(*ZoneData)
}

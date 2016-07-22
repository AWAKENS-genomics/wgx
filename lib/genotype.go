package wgx

import (
	"bytes"
	"encoding/json"
	// log "github.com/Sirupsen/logrus"
	"github.com/brentp/bix"
	"github.com/brentp/irelate/interfaces"
)

type Genotype struct {
	Chrom    string   `json:"chrom"`
	Position int      `json:"position"`
	Id       string   `json:"id"`
	Genotype []string `json:"genotype"`
	Alleles  []string `json:"alleles"`
}

type Genotypes struct {
	SampleName string     `json:"sampleName"`
	Genotypes  []Genotype `json:"genotypes"`
}

func (genotypes *Genotypes) AddGenotype(genotype Genotype) []Genotype {
	genotypes.Genotypes = append(genotypes.Genotypes, genotype)
	return genotypes.Genotypes
}

type Location struct {
	chrom string
	start int
	end   int
}

func (s Location) Chrom() string {
	return s.chrom
}
func (s Location) Start() uint32 {
	return uint32(s.start)
}
func (s Location) End() uint32 {
	return uint32(s.end)
}

func NewLocation(chrom string, start int, end int) Location {
	return Location{chrom, start, end}
}

func QueryGenotypes(f string, locs []Location) ([]byte, error) {
	var genotypes Genotypes
	var sampleName string

	tbx, err := bix.New(f)
	if err != nil {
		return nil, err
	}

	vr := tbx.VReader

	for i := range locs {
		vals, _ := tbx.Query(locs[i])

		// FIXME: Assert one record for one query
		v, err := vals.Next()
		if err != nil {
			return nil, err
		}

		// Parse sample names
		line := []byte(v.(interfaces.IVariant).String())
		fields := makeFields(line)
		variant := vr.Parse(fields)
		vr.Header.ParseSamples(variant)
		sampleNames := vr.Header.SampleNames
		samples := variant.Samples

		chrom := v.(interfaces.IPosition).Chrom()
		pos := v.(interfaces.IPosition).End()
		id_ := v.(interfaces.IVariant).Id()
		// info_ := v.(interfaces.IVariant).Info()

		// Parse alleles
		ref := v.(interfaces.IVariant).Ref()
		alt := v.(interfaces.IVariant).Alt()
		alleles := []string{}
		alleles = append(alleles, ref)
		alleles = append(alleles, alt...)

		// Get genotypes
		idx := 0
		sample := samples[idx]
		sampleName = sampleNames[idx]

		genotype := []string{}
		gt := sample.GT
		for j := range gt {
			genotype = append(genotype, alleles[gt[j]])
		}

		genotypes.AddGenotype(Genotype{chrom, int(pos), id_, genotype, alleles})
	}

	tbx.Close()

	genotypes.SampleName = sampleName
	response, err := json.Marshal(genotypes)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// copied from github.com/brentp/bix
func makeFields(line []byte) [][]byte {
	fields := make([][]byte, 9)
	copy(fields[:8], bytes.SplitN(line, []byte{'\t'}, 8))
	s := 0
	for i, f := range fields {
		if i == 7 {
			break
		}
		s += len(f) + 1
	}
	e := bytes.IndexByte(line[s:], '\t')
	if e == -1 {
		e = len(line)
	} else {
		e += s
	}

	fields[7] = line[s:e]
	if len(line) > e+1 {
		fields[8] = line[e+1:]
	} else {
		fields = fields[:8]
	}

	return fields
}

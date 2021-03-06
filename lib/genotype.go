package awtk

import (
	"bytes"
	// log "github.com/Sirupsen/logrus"
	"github.com/brentp/bix"
	"github.com/brentp/irelate/interfaces"
)

type Genotype struct {
	Chrom     string   `json:"chrom"`
	Position  int      `json:"position"`
	SnpId     string   `json:"snpId"`
	Genotype  []string `json:"genotype"`
	Alleles   []string `json:"alleles"`
	Reference string   `json:"reference"`
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

type Sequence struct {
	Chrom      string   `json:"chrom"`
	Start      int      `json:"start"`
	End        int      `json:"end"`
	Reference  []string `json:"reference"`
	Haplotype1 []string `json:"haplotype_1"`
	Haplotype2 []string `json:"haplotype_2"`
}

func QueryGenotypes(f string, idx int, locs []Location) (Genotypes, error) {
	var genotypes Genotypes
	var sampleName string

	tbx, err := bix.New(f)
	if err != nil {
		return Genotypes{}, err
	}

	vr := tbx.VReader

	for i := range locs {
		vals, _ := tbx.Query(locs[i])

		for {
			v, err := vals.Next()

			if err != nil {
				break
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
			snpId := v.(interfaces.IVariant).Id()
			// info := v.(interfaces.IVariant).Info()

			// Parse alleles
			ref := v.(interfaces.IVariant).Ref()
			alt := v.(interfaces.IVariant).Alt()
			alleles := []string{}
			alleles = append(alleles, ref)

			if len(alt) == 1 && alt[0] == "." {
				// no ALTs (= ALT is ["."])
			} else {
				alleles = append(alleles, alt...)
			}

			// Get genotypes of 1st sample
			sample := samples[idx]
			sampleName = sampleNames[idx]

			genotype := []string{}
			gt := sample.GT

			for j := range gt {
				if gt[j] == -1 {
					// no GTs (= GT is missing value: -1)
					genotype = append(genotype, ".")
				} else {
					genotype = append(genotype, alleles[gt[j]])
				}

			}

			genotypes.AddGenotype(Genotype{chrom,
				int(pos),
				snpId,
				genotype,
				alleles,
				ref})
		}
	}

	tbx.Close()

	genotypes.SampleName = sampleName

	return genotypes, nil
}

// Convert Genotypes to Sequence
func Genotypes2Sequence(gts Genotypes, locs []Location) (Sequence, error) {
	loc := locs[0]
	chrom := loc.Chrom()
	start := int(loc.Start()) + 1
	end := int(loc.End())
	seq := Sequence{chrom, start, end, []string{}, []string{}, []string{}}

	// ["N", "N", ..., "N"]
	for i := start; i <= end; i++ {
		seq.Reference = append(seq.Reference, "N")
		seq.Haplotype1 = append(seq.Haplotype1, "N")
		seq.Haplotype2 = append(seq.Haplotype2, "N")
	}

	// ["G", "A", ..., "N"]
	genotypes := gts.Genotypes
	for j := range genotypes {
		genotype := genotypes[j]
		gt := genotype.Genotype
		idx := genotype.Position - start // 100 - 100 = 0, 101 - 100 = 1, ...
		seq.Reference[idx] = genotype.Reference
		seq.Haplotype1[idx] = gt[0]
		seq.Haplotype2[idx] = gt[1]
	}
	return seq, nil
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

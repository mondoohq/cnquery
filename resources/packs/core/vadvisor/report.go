package vadvisor

// Determine all Cves of all Advisories
func (r *VulnReport) Cves() []*CVE {
	cveMap := map[string]*CVE{}

	for i := range r.Advisories {
		advisory := r.Advisories[i]
		for j := range advisory.Cves {
			cve := advisory.Cves[j]
			cveMap[cve.ID] = cve
		}
	}

	cveList := []*CVE{}
	for _, v := range cveMap {
		cveList = append(cveList, v)
	}
	return cveList
}

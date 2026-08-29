package trivy

import data.lib.trivy

default ignore = false

ignore {
    input.VulnerabilityID == "CVE-2024-24786"
}

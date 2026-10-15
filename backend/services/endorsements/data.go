package endorsements

// The record itself.
//
// Every person here is real and every line is drawn from their English
// Wikipedia article, cited per person in Source. Where that article gave
// a year or a range, Period carries it; where it gave none, Period is
// empty rather than guessed. Nothing has been rounded up, totalled, or
// turned into a figure the source did not state.
//
// Photographs are freely licensed, downloaded from Wikimedia Commons and
// served locally so the record does not depend on a hotlink. Each one
// credits its photographer and licence, which is a condition of all of
// these licences and not a courtesy.
var people = []Person{
	{
		ID: "virat-kohli", Name: "Virat Kohli", Role: "Cricketer", Initials: "VK",
		Field:  "Cricket",
		Source: "https://en.wikipedia.org/wiki/Virat_Kohli",
		Photo: Photo{
			URL: "/celebs/virat-kohli.jpg", Artist: "Prime Minister's Office",
			License: "GODL-India", LicenseURL: "https://data.gov.in/government-open-data-license-india",
			Source: "https://commons.wikimedia.org/wiki/File:Virat_Kohli_during_the_India_vs_Aus_4th_Test_match_at_Narendra_Modi_Stadium_on_09_March_2023.jpg",
		},
		Deals: []Deal{
			{Brand: "Puma", Period: "2017–2025", Note: "Eight-year deal reported at about ₹110 crore", Kind: KindEndorsement},
			{Brand: "MRF", Note: "Bat sponsorship", Kind: KindEndorsement},
			{Brand: "Vivo", Period: "2021–", Note: "Brand ambassador", Kind: KindAmbassador},
			{Brand: "Mobile Premier League", Period: "2019–", Note: "Brand ambassador", Kind: KindAmbassador},
			{Brand: "Audi", Note: "Q7 commercial", Kind: KindEndorsement},
			{Brand: "Agilitas", Period: "2025–", Note: "Joined as investor after the Puma deal concluded", Kind: KindStake},
		},
	},
	{
		ID: "alia-bhatt", Name: "Alia Bhatt", Role: "Actor", Initials: "AB",
		Field:  "Film",
		Source: "https://en.wikipedia.org/wiki/Alia_Bhatt",
		Photo: Photo{
			URL: "/celebs/alia-bhatt.jpg", Artist: "Gabriel Hutchinson",
			License: "CC BY-SA 4.0", LicenseURL: "https://creativecommons.org/licenses/by-sa/4.0",
			Source: "https://commons.wikimedia.org/wiki/File:Alia_Bhatt_attends_at_the_2026_Cannes_Film_Festival_(cropped)_(cropped).jpg",
		},
		Deals: []Deal{
			{Brand: "Gucci", Period: "2023–", Note: "First Indian global ambassador for the house", Kind: KindAmbassador},
			{Brand: "L'Oréal", Period: "2024–", Note: "Global ambassador", Kind: KindAmbassador},
			{Brand: "Levi's", Period: "2025–", Note: "Global ambassador", Kind: KindAmbassador},
			{Brand: "Maybelline", Kind: KindEndorsement},
			{Brand: "Garnier", Kind: KindEndorsement},
			{Brand: "Coca-Cola", Kind: KindEndorsement},
			{Brand: "Ed-a-Mamma", Period: "2022–", Note: "Founder, not an endorsement", Kind: KindStake},
			{Brand: "Nykaa", Note: "Investor", Kind: KindStake},
		},
	},
	{
		ID: "salman-khan", Name: "Salman Khan", Role: "Actor", Initials: "SK",
		Field:  "Film",
		Source: "https://en.wikipedia.org/wiki/Salman_Khan",
		Photo: Photo{
			URL: "/celebs/salman-khan.jpg", Artist: "Bollywood Hungama",
			License: "CC BY 3.0", LicenseURL: "https://creativecommons.org/licenses/by/3.0",
			Source: "https://commons.wikimedia.org/wiki/File:Salman_Khan_in_2023_(cropped).jpg",
		},
		Deals: []Deal{
			{Brand: "Pepsi", Kind: KindEndorsement},
			{Brand: "Suzuki", Note: "Motorcycles", Kind: KindEndorsement},
			{Brand: "Relaxo Hawaii", Note: "Footwear", Kind: KindEndorsement},
			{Brand: "Wheel", Note: "Detergent", Kind: KindEndorsement},
			{Brand: "CP PLUS", Note: "Security and surveillance", Kind: KindEndorsement},
			{Brand: "Astral", Note: "Plumbing and drainage", Kind: KindAmbassador},
			{Brand: "Splash", Note: "Dubai-based fashion label", Kind: KindAmbassador},
			{Brand: "Thums Up", Period: "2002", Kind: KindEndorsement},
			{Brand: "Mountain Dew", Period: "until December 2010", Kind: KindEndorsement},
			{Brand: "Yatra", Note: "Endorser and shareholder", Kind: KindStake},
		},
	},
	{
		ID: "deepika-padukone", Name: "Deepika Padukone", Role: "Actor", Initials: "DP",
		Field:  "Film",
		Source: "https://en.wikipedia.org/wiki/Deepika_Padukone",
		Photo: Photo{
			URL: "/celebs/deepika-padukone.jpg", Artist: "Narendra Modi",
			License: "CC BY 3.0", LicenseURL: "https://creativecommons.org/licenses/by/3.0",
			Source: "https://commons.wikimedia.org/wiki/File:Deepika_Padukone_2025_(1).png",
		},
		Deals: []Deal{
			{Brand: "Louis Vuitton", Period: "2020–", Note: "First Indian house ambassador", Kind: KindAmbassador},
			{Brand: "Cartier", Period: "2022–", Kind: KindAmbassador},
			{Brand: "L'Oréal Paris", Kind: KindEndorsement},
			{Brand: "Maybelline", Kind: KindEndorsement},
			{Brand: "Tissot", Kind: KindEndorsement},
			{Brand: "Coca-Cola", Kind: KindEndorsement},
		},
	},
	{
		ID: "ranveer-singh", Name: "Ranveer Singh", Role: "Actor", Initials: "RS",
		Field:  "Film",
		Source: "https://en.wikipedia.org/wiki/Ranveer_Singh",
		Photo: Photo{
			URL: "/celebs/ranveer-singh.jpg", Artist: "Bollywood Hungama",
			License: "CC BY 3.0", LicenseURL: "https://creativecommons.org/licenses/by/3.0",
			Source: "https://commons.wikimedia.org/wiki/File:Ranveer_Singh_in_2023_(1)_(cropped).jpg",
		},
		Deals: []Deal{
			{Brand: "Adidas", Kind: KindEndorsement},
			{Brand: "Head & Shoulders", Kind: KindEndorsement},
			{Brand: "Jack & Jones", Kind: KindEndorsement},
			{Brand: "Ching's Secret", Kind: KindEndorsement},
			{Brand: "Thums Up", Kind: KindEndorsement},
			{Brand: "MakeMyTrip", Kind: KindEndorsement},
			{Brand: "Sugar Cosmetics", Note: "Investor", Kind: KindStake},
		},
	},
	{
		ID: "bhuvan-bam", Name: "Bhuvan Bam", Role: "Content creator", Initials: "BB",
		Field:  "Digital",
		Source: "https://en.wikipedia.org/wiki/Bhuvan_Bam",
		Photo: Photo{
			URL: "/celebs/bhuvan-bam.jpg", Artist: "Bollywood Hungama",
			License: "CC BY 3.0", LicenseURL: "https://creativecommons.org/licenses/by/3.0",
			Source: "https://commons.wikimedia.org/wiki/File:Bhuvan_Bam_at_Myntra's_Creator_Fest_2023_event_(cropped).jpg",
		},
		Deals: []Deal{
			{Brand: "Lenskart", Note: "Their first male brand ambassador", Kind: KindAmbassador},
			{Brand: "Ola Electric", Note: "One of its celebrity brand ambassadors", Kind: KindAmbassador},
			{Brand: "Beyoung", Period: "2024–", Note: "Their first brand ambassador", Kind: KindAmbassador},
			{Brand: "Faboom", Note: "Brand ambassador", Kind: KindAmbassador},
			{Brand: "Puma", Period: "2020", Note: "Grazia India cover collaboration", Kind: KindEndorsement},
		},
	},
	{
		ID: "prajakta-koli", Name: "Prajakta Koli", Role: "Content creator", Initials: "PK",
		Field:  "Digital",
		Source: "https://en.wikipedia.org/wiki/Prajakta_Koli",
		Photo: Photo{
			URL: "/celebs/prajakta-koli.jpg", Artist: "Bollywood Hungama",
			License: "CC BY 3.0", LicenseURL: "https://creativecommons.org/licenses/by/3.0",
			Source: "https://commons.wikimedia.org/wiki/File:Prajakta_koli_in_2023.jpj.png",
		},
		Deals: []Deal{
			{Brand: "WhatsApp", Period: "2018", Note: "One of three national television commercials", Kind: KindEndorsement},
			{Brand: "Netflix", Period: "2020–", Note: "Mismatched, and Single Papa in 2025", Kind: KindEndorsement},
			{Brand: "YouTube", Period: "2018–", Note: "Creators for Change, and the Original Pretty Fit", Kind: KindEndorsement},
		},
	},
	{
		ID: "kusha-kapila", Name: "Kusha Kapila", Role: "Content creator", Initials: "KK",
		Field:  "Digital",
		Source: "https://en.wikipedia.org/wiki/Kusha_Kapila",
		Photo: Photo{
			URL: "/celebs/kusha-kapila.jpg", Artist: "Bollywood Hungama",
			License: "CC BY 3.0", LicenseURL: "https://creativecommons.org/licenses/by/3.0",
			Source: "https://commons.wikimedia.org/wiki/File:Kusha_Kapila_snapped_at_the_red_carpet_of_ET_Inspiring_Women_Leaders_Conclave.jpg",
		},
		Deals: []Deal{
			{Brand: "Orient Electric", Period: "2025", Note: "Premium lighting campaign, alongside MS Dhoni", Kind: KindEndorsement},
		},
	},
	{
		ID: "tamannaah-bhatia", Name: "Tamannaah Bhatia", Role: "Actor", Initials: "TB",
		Field:  "Film",
		Source: "https://en.wikipedia.org/wiki/Tamannaah",
		Photo: Photo{
			URL: "/celebs/tamannaah-bhatia.jpg", Artist: "Bollywood Hungama",
			License: "CC BY 3.0", LicenseURL: "https://creativecommons.org/licenses/by/3.0",
			Source: "https://commons.wikimedia.org/wiki/File:Tamannaah_Bhatia_Lakme_Fashion_Week_2017.jpg",
		},
		Deals: []Deal{
			{Brand: "Havmor", Period: "2025", Note: "Brand ambassador — \"Soooo Tasty, You Wanna Havmoorrrr!\"", Kind: KindAmbassador},
			{Brand: "Mother Dairy", Period: "2026", Note: "Flavoured milk campaign", Kind: KindEndorsement},
			{Brand: "Guess Watches", Period: "2026", Note: "Brand ambassador, FW26 watches collection", Kind: KindAmbassador},
			{Brand: "Shiseido", Period: "2025", Note: "Ultimune Power Infusing Serum campaign", Kind: KindEndorsement},
		},
	},
	{
		ID: "ms-dhoni", Name: "MS Dhoni", Role: "Cricketer", Initials: "MD",
		Field:  "Cricket",
		Source: "https://en.wikipedia.org/wiki/MS_Dhoni",
		Photo: Photo{
			URL: "/celebs/ms-dhoni.jpg", Artist: "RIGI",
			License: "CC BY 3.0", LicenseURL: "https://creativecommons.org/licenses/by/3.0",
			Source: "https://commons.wikimedia.org/wiki/File:MS_Dhoni_(Prabhav_'23_-_RiGI_2023).jpg",
		},
		Deals: []Deal{
			{Brand: "SEVEN", Period: "2016–", Note: "Co-owner and brand ambassador", Kind: KindStake},
			{Brand: "CARS24", Period: "2019–", Note: "Investor and brand ambassador", Kind: KindStake},
			{Brand: "Shaka Harry", Period: "2022–", Note: "Investor in the plant-protein company", Kind: KindStake},
		},
	},
}

// All returns the record. A copy of the slice header is enough: nothing
// here is written after init, and handing out the package's own slice
// would let a caller reorder the record for everyone.
func All() []Person {
	out := make([]Person, len(people))
	copy(out, people)
	return out
}

// Find returns one person by id, and whether they are in the record.
func Find(id string) (Person, bool) {
	for _, p := range people {
		if p.ID == id {
			return p, true
		}
	}
	return Person{}, false
}

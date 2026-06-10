package v1

import (
	"strings"

	"github.com/tobiasehlert/teslamateapi/pkg/dto"
)

type localizedVINText struct {
	en string
	zh string
}

func (t localizedVINText) value(lang string) string {
	if normalizeVINLanguage(lang) == "zh" && t.zh != "" {
		return t.zh
	}
	return t.en
}

var teslaVINDescriptions = struct {
	wmi              map[string]localizedVINText
	lineSeries       map[byte]localizedVINText
	modelYBodyType   map[byte]localizedVINText
	modelYRestraint  map[byte]localizedVINText
	modelYMotorDrive map[byte]localizedVINText
	productionYear   map[byte]int
	plant            map[byte]localizedVINText
}{
	wmi: map[string]localizedVINText{
		"5YJ": {en: "Tesla, Inc. (United States)", zh: "特斯拉公司（美国）"},
		"7SA": {en: "Tesla, Inc. (United States)", zh: "特斯拉公司（美国）"},
		"LRW": {en: "Tesla China (Giga Shanghai)", zh: "特斯拉中国（上海超级工厂）"},
		"XP7": {en: "Tesla Manufacturing Brandenburg SE (Germany)", zh: "特斯拉勃兰登堡制造公司（德国）"},
	},
	lineSeries: map[byte]localizedVINText{
		'S': {en: "Model S", zh: "Model S"},
		'3': {en: "Model 3", zh: "Model 3"},
		'X': {en: "Model X", zh: "Model X"},
		'Y': {en: "Model Y", zh: "Model Y"},
	},
	modelYBodyType: map[byte]localizedVINText{
		'G': {en: "MPV 5 DR / LHD", zh: "MPV 5 门 / 左舵"},
		'H': {en: "MPV 5 DR / RHD", zh: "MPV 5 门 / 右舵"},
	},
	modelYRestraint: map[byte]localizedVINText{
		'A': {en: "Type 2 Manual seatbelts (FR, SR*3, TR*2) with front Airbags, PODS, side Inflatable restraints, Knee Airbags", zh: "2 型手动安全带（前排、第二排 3 个、第三排 2 个），带前排安全气囊、乘员检测系统、侧面充气约束系统和膝部安全气囊"},
		'D': {en: "Type 2 Manual seatbelts (FR, SR*3) with front Airbags, PODS, side Inflatable restraints, Knee Airbags", zh: "2 型手动安全带（前排、第二排 3 个），带前排安全气囊、乘员检测系统、侧面充气约束系统和膝部安全气囊"},
		'C': {en: "Type 2 Manual seatbelts with front Airbags, PODS, side Inflatable restraints", zh: "2 型手动安全带，带前排安全气囊、乘员检测系统和侧面充气约束系统"},
	},
	modelYMotorDrive: map[byte]localizedVINText{
		'D': {en: "Single Motor Standard", zh: "单电机标准版"},
		'E': {en: "Dual Motor Standard", zh: "双电机标准版"},
		'F': {en: "Dual Motor Performance (3DU 800A)", zh: "双电机性能版（3DU 800A）"},
		'J': {en: "Single Motor Standard", zh: "单电机标准版"},
		'K': {en: "Dual Motor Standard", zh: "双电机标准版"},
		'L': {en: "Dual Motor Performance", zh: "双电机性能版"},
		'R': {en: "Single Motor Standard (3DU 600A)", zh: "单电机标准版（3DU 600A）"},
		'S': {en: "Single Motor Standard (DUB 600A)", zh: "单电机标准版（DUB 600A）"},
	},
	productionYear: map[byte]int{
		'A': 2010,
		'B': 2011,
		'C': 2012,
		'D': 2013,
		'E': 2014,
		'F': 2015,
		'G': 2016,
		'H': 2017,
		'J': 2018,
		'K': 2019,
		'L': 2020,
		'M': 2021,
		'N': 2022,
		'P': 2023,
		'R': 2024,
		'S': 2025,
		'T': 2026,
		'V': 2027,
		'W': 2028,
		'X': 2029,
		'Y': 2030,
	},
	plant: map[byte]localizedVINText{
		'A': {en: "Tesla Austin, TX (USA)", zh: "特斯拉奥斯汀工厂（美国得克萨斯州）"},
		'B': {en: "Tesla Berlin (Germany)", zh: "特斯拉柏林工厂（德国）"},
		'C': {en: "Tesla China (Giga Shanghai)", zh: "特斯拉中国（上海超级工厂）"},
		'F': {en: "Tesla Fremont, CA (USA)", zh: "特斯拉弗里蒙特工厂（美国加利福尼亚州）"},
	},
}

func decodeTeslaVIN(raw, lang string) *dto.V1CarVINDetails {
	vin := strings.ToUpper(strings.TrimSpace(raw))
	if len(vin) != 17 {
		return nil
	}

	wmi := vin[:3]
	manufacturer, ok := teslaVINDescriptions.wmi[wmi]
	if !ok {
		return nil
	}

	details := &dto.V1CarVINDetails{
		WMI:            wmi,
		Manufacturer:   manufacturer.value(lang),
		CheckDigit:     vin[8:9],
		YearType:       teslaVINYearType(wmi, lang),
		SequenceNumber: vin[11:17],
	}

	if lineSeries, ok := teslaVINDescriptions.lineSeries[vin[3]]; ok {
		details.LineSeries = lineSeries.value(lang)
	}
	if year, ok := teslaVINDescriptions.productionYear[vin[9]]; ok {
		details.ProductionYear = year
	}
	if plant, ok := teslaVINDescriptions.plant[vin[10]]; ok {
		details.Plant = plant.value(lang)
	}

	if vin[3] == 'Y' {
		decodeTeslaModelYVINDetails(vin, wmi, lang, details)
	}

	return details
}

func decodeTeslaModelYVINDetails(vin, wmi, lang string, details *dto.V1CarVINDetails) {
	if bodyType, ok := teslaVINDescriptions.modelYBodyType[vin[4]]; ok {
		details.BodyType = bodyType.value(lang)
	}
	if restraintSystem, ok := teslaVINDescriptions.modelYRestraint[vin[5]]; ok {
		details.RestraintSystem = restraintSystem.value(lang)
	}
	if fuelType, ok := teslaModelYVINFuelType(wmi, vin[6]); ok {
		details.FuelType = fuelType.value(lang)
	}
	if motorDriveUnit, ok := teslaVINDescriptions.modelYMotorDrive[vin[7]]; ok {
		details.MotorDriveUnit = motorDriveUnit.value(lang)
	}
}

func teslaVINYearType(wmi, lang string) string {
	if wmi == "LRW" || wmi == "XP7" {
		return localizedVINText{en: "calendar_year", zh: "日历年"}.value(lang)
	}
	return localizedVINText{en: "model_year", zh: "车型年"}.value(lang)
}

func teslaModelYVINFuelType(wmi string, code byte) (localizedVINText, bool) {
	switch code {
	case 'E':
		if wmi == "LRW" {
			return localizedVINText{en: "Ternary System Li-ion battery", zh: "三元锂电池"}, true
		}
		return localizedVINText{en: "Electric", zh: "纯电动"}, true
	case 'F':
		return localizedVINText{en: "Lithium Iron Phosphate Battery", zh: "磷酸铁锂电池"}, wmi == "LRW"
	default:
		return localizedVINText{}, false
	}
}

func normalizeVINLanguage(lang string) string {
	normalized := strings.ToLower(strings.TrimSpace(lang))
	if dot := strings.IndexByte(normalized, '.'); dot >= 0 {
		normalized = normalized[:dot]
	}
	normalized = strings.ReplaceAll(normalized, "_", "-")
	if strings.HasPrefix(normalized, "zh") {
		return "zh"
	}
	return "en"
}

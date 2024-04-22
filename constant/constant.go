package constant

const (
	// SepFlag 竖线分割符
	SepFlag = "|"
	// Comma 逗号
	Comma = ","
	// Hyphen 连字符
	Hyphen = "-"
	// Colon 冒号
	Colon = ":"
	// CommConfigPath 通用配置文件路径
	CommConfigPath = "comm.yaml"

	// TrueFlag 表示真的数值
	TrueFlag = 1
	// FalseFlag 表示假的数值
	FalseFlag = 0

	TypeAdd   = 1
	TypeMinus = -1
)

const (
	MySQLDrive = "mysql" // mysql驱动

	SuccessString = "success"

	DefaultPage  = 0  // 默认的页
	DefaultLimit = 10 // 默认的页大小

	MethodPost = "POST"
	MethodGet  = "GET"

	RetryTimes = 3
)

type Weekday int

const (
	Sunday Weekday = iota
	Monday
	Tuesday
	Wednesday
	Thursday
	Friday
	Saturday
)

var WeekDayMap = map[string]string{
	"Monday":    "周一",
	"Tuesday":   "周二",
	"Wednesday": "周三",
	"Thursday":  "周四",
	"Friday":    "周五",
	"Saturday":  "周六",
	"Sunday":    "周日",
}

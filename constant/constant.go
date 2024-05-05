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

var ErrorMap = map[int]string{
	10000: "成功",
	10001: "请求的数据为空",
	10002: "请求所需的参数部分缺失",
	10003: "请求的参数存在不合法的情况",
	10004: "登录失效",
	10005: "信息不匹配",
	10006: "权限不匹配",

	20000: "查询错误",
	20001: "Scan错误",
	20002: "插入错误",
	20003: "删除错误",
	20004: "更新错误",
	20005: "计数错误",
	20006: "获取影响行数错误",
	20007: "插入时Prepare的错误",
	20008: "插入时Exec的错误",
	20009: "事务操作时的错误",
	20010: "获取ID错误",

	30000: "类型转换错误",
	30001: "日期类型的错误",
	30002: "未知错误",
	30003: "该类型已存在",
	30004: "缓存相关错误",

	40000: "http请求DoRequest错误",
	40001: "http请求NewRequest错误",
	40002: "http请求get错误",
	40003: "http请求post错误",

	50000: "json反序列化错误",
	50001: "json序列化错误",

	60000: "读取文件错误",
	60001: "文件不存在",
	60002: "打开文件错误",
	60003: "文件写入错误",
	60004: "删除文件错误",
	60005: "创建文件错误",
}

package constant

// 返回1xxxx -> 请求成功
const (
	Success          = 10000 // 请求成功的时候的返回值
	DataEmpty        = 10001 // 请求的数据为空时候的返回值
	ParameterMiss    = 10002 // 请求所需的参数部分缺失
	ParameterInvalid = 10003 // 请求的参数存在不合法的情况
)

// 返回2xxxx -> MySQL错误
const (
	MySQLQueryError       = 20000 // MySQL的查询错误
	MySQLScanError        = 20001 // MySQL在Scan时候的错误
	MySQLInsertError      = 20002 // MySQL的插入错误
	MySQLDeleteError      = 20003 // MySQL的删除错误
	MySQLUpdateError      = 20004 // MySQL的更新错误
	MySQLCountError       = 20004 // MySQL的计数错误
	MySQLAffectRowError   = 20005 // MySQL获取影响行数错误
	MySQLPrepareError     = 20006 // MySQL 插入时Prepare的错误
	MySQLExecError        = 20007 // MySQL 插入时Exec的错误
	MySQLTransactionError = 20007 // MySQL 事务操作时的错误
)

// 返回3xxxx -> 其他类型的错误
const (
	StrconvAtoiError = 30000 // 类型转换错误（主要是string转int）
	ParseTimeError   = 30001 // 日期类型的错误
	ReadFileError    = 30002 // 读取文件错误
	UnknowError      = 30003 // 未知错误
	UnknowOwner      = 30004 // 不是球馆管理员返回码
	InfoNotMatch     = 30005 // 信息不匹配
	ModelIsExist     = 30006 // 该类型已存在
)

// 返回4xxxx -> json错误
const (
	JSONUnmarshalError = 40000 // json反序列化错误
	JSONMarshalError   = 40001 // json序列化错误
)

// 返回8xxxx -> HTTP错误
const (
	HTTPDoReqError  = 80000
	HTTPNewReqError = 80001
	HTTPGetError    = 80002
	HTTPPostError   = 80003
)

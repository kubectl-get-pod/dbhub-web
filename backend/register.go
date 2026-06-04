package main

import (
	"dbhub-web/connector"
	"dbhub-web/connector/mssql"
	"dbhub-web/connector/mysql"
	"dbhub-web/connector/oracle"
	"dbhub-web/connector/postgres"
)

func init() {
	// 注册所有数据库插件
	connector.GlobalRegistry.Register(mysql.PluginName, mysql.New())
	connector.GlobalRegistry.Register(postgres.PluginName, postgres.New())
	connector.GlobalRegistry.Register(oracle.PluginName, oracle.New())
	connector.GlobalRegistry.Register(mssql.PluginName, mssql.New())
}

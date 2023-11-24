package clusterctl

import (
	"context"
	"database/sql"
	dorisv1alpha1 "doris-operator/api/v1alpha1"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"

	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	_ "github.com/go-sql-driver/mysql"
)

func initFeConnect(ctx context.Context, feLeaderip string) (*sql.DB, error) {
	log := ctrllog.FromContext(ctx)

	dsn := "root:@tcp(" + feLeaderip + ":9030)/?charset=utf8mb4&parseTime=True"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Error(err, err.Error())
		return db, err
	}
	// reconnect fe
	for i := 0; i < 100; i++ {
		time.Sleep(10 * time.Second)
		err = db.Ping()
		if err == nil {
			break
		}
		if i == 99 {
			return db, err
		}
	}
	return db, err
}

func querySql(ctx context.Context, feLeaderip string, sqlStr string) *sql.Rows {
	log := ctrllog.FromContext(ctx)

	db, err := initFeConnect(ctx, feLeaderip)
	if err != nil {
		log.Error(err, err.Error())
	}
	log.Info(sqlStr)
	rows, err := db.Query(sqlStr)
	if err != nil {
		log.Error(err, err.Error())
	}
	return rows
}

func sqlResultConvertStruct(ctx context.Context, rows *sql.Rows) []map[string]interface{} {
	log := ctrllog.FromContext(ctx)
	columns, _ := rows.Columns()
	count := len(columns)
	values := make([]interface{}, count)
	valuePtr := make([]interface{}, count)

	var Values []map[string]interface{}

	for rows.Next() {
		Value := make(map[string]interface{}, count)
		for i, _ := range columns {
			valuePtr[i] = &values[i]
		}
		err := rows.Scan(valuePtr...)
		if err != nil {
			log.Error(err, err.Error())
		}
		for i, col := range columns {
			var v interface{}
			val := values[i]
			b, ok := val.([]byte)
			if ok {
				v = string(b)
			} else {
				v = val
			}
			Value[col] = v
		}
		Values = append(Values, Value)
	}
	return Values
}

func (r *ClusterReconciler) feRegistryBe(ctx context.Context, feLeaderip string, beip string, cluster *dorisv1alpha1.Cluster, newBe *dorisv1alpha1.Be) {
	querySqlStr := "ALTER SYSTEM ADD BACKEND " + "\"" + beip + ":9050" + "\""
	querySql(ctx, feLeaderip, querySqlStr)
	r.Recorder.Event(cluster, corev1.EventTypeNormal, "Registry be ", fmt.Sprintf("name is %s", newBe.Name))
}

func (r *ClusterReconciler) deleteBe(ctx context.Context, feLeaderip string, beIP string) {
	querySqlStr := "ALTER SYSTEM DECOMMISSION BACKEND " + "\"" + beIP + ":9050" + "\""
	querySql(ctx, feLeaderip, querySqlStr)
}

func (r *ClusterReconciler) addFe(ctx context.Context, feLeaderip string, feip string) {
	querySqlStr := "ALTER SYSTEM ADD FOLLOWER " + "\"" + feip + ":9010" + "\""
	querySql(ctx, feLeaderip, querySqlStr)
}

func (r *ClusterReconciler) deleteFe(ctx context.Context, feLeaderip string, feip string) {
	querySqlStr := "ALTER SYSTEM DROP FOLLOWER " + "\"" + feip + ":9010" + "\""
	querySql(ctx, feLeaderip, querySqlStr)
}

func showFrontends(ctx context.Context, feLeaderip string) []map[string]interface{} {
	querySqlStr := "SHOW PROC " + "\"" + "/frontends" + "\""
	rows := querySql(ctx, feLeaderip, querySqlStr)
	return sqlResultConvertStruct(ctx, rows)
}

func showBackends(ctx context.Context, feLeaderip string) []map[string]interface{} {
	querySqlStr := "SHOW PROC " + "\"" + "/backends" + "\""
	rows := querySql(ctx, feLeaderip, querySqlStr)
	return sqlResultConvertStruct(ctx, rows)
}

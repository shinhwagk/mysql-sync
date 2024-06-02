package main

// import "github.com/go-mysql-org/go-mysql/client"

// // // Connect MySQL at 127.0.0.1:3306, with user root, an empty password and database test
// // conn, _ := client.Connect("127.0.0.1:3306", "root", "", "test")
// type MysqlSyncTarget struct {
// 	con client.Conn
// }

// // Or to use SSL/TLS connection if MySQL server supports TLS
// //conn, _ := client.Connect("127.0.0.1:3306", "root", "", "test", func(c *Conn) {c.UseSSL(true)})

// // Or to set your own client-side certificates for identity verification for security
// //tlsConfig := NewClientTLSConfig(caPem, certPem, keyPem, false, "your-server-name")
// //conn, _ := client.Connect("127.0.0.1:3306", "root", "", "test", func(c *Conn) {c.SetTLSConfig(tlsConfig)})

// conn.Ping()

// // Insert
// r, _ := conn.Execute(`insert into table (id, name) values (1, "abc")`)

// // Get last insert id
// println(r.InsertId)
// // Or affected rows count
// println(r.AffectedRows)

// // Select
// r, err := conn.Execute(`select id, name from table where id = 1`)

// // Close result for reuse memory (it's not necessary but very useful)
// defer r.Close()

// // Handle resultset
// v, _ := r.GetInt(0, 0)
// v, _ = r.GetIntByName(0, "id")

// // Direct access to fields
// for _, row := range r.Values {
// 	for _, val := range row {
// 		_ = val.Value() // interface{}
// 		// or
// 		if val.Type == mysql.FieldValueTypeFloat {
// 			_ = val.AsFloat64() // float64
// 		}
// 	}
// }

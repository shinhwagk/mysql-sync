{
    "Job": {
        "ID": "dtle-demo",
        "Datacenters": [
            "dc1"
        ],
        "TaskGroups": [
            {
                "Name": "src",
                "Tasks": [
                    {
                        "Name": "src",
                        "Driver": "dtle",
                        "Config": {
                            "Gtid": "",
                            "ReplicateDoDb": [
                                {
                                    "TableSchema": "test"
                                }
                            ],
                            "SrcConnectionConfig": {
                                "Host": "db1",
                                "Port": 3306,
                                "User": "root",
                                "Password": "root_password"
                            },
                            "DestConnectionConfig": {
                                "Host": "db2",
                                "Port": 3306,
                                "User": "root",
                                "Password": "root_password"
                            }
                        }
                    }
                ]
            },
            {
                "Name": "dest",
                "Tasks": [
                    {
                        "Name": "dest",
                        "Driver": "dtle",
                        "Config": {
                            "DestType": "mysql"
                        }
                    }
                ]
            }
        ]
    }
}
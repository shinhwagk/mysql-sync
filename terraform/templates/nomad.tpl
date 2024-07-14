job "mysqlsync-${repl.name}" {
  region = "dc1"
  datacenters = ${jsonencode(all_datacenters)}
  group "repl-${repl.name}" {
    datacenters = ${jsonencode(repl.nomad.datacenters)}
    network {
      mode = "bridge"
    }
    service {
      name = "repl"
      port = "9998"
      connect {
        sidecar_service {}
      }
    }
    template {
      data        = "{{ key \"${config_key}\" }}"
      destination = "local/config.yml"
    }
    task "repl-${repl.name}" {
      driver = "docker"
      config {
        image = "${repl.nomad.image}"
        args  = "-repl"
        ports = ["9998"]
        volumes = [
          "local/config.yml:/etc/mysqlsync/config.yml"
        ]
      }
    }
  }
%{ for dest_name in dest_keys ~}
  group "dest_${dest_name}" {
    datacenters = ${jsonencode(dests[dest_name].nomad.datacenters)}
    service {
      connect {
        sidecar_service {
          proxy {
            upstreams {
              destination_name = "repl"
              local_bind_port  = 9998
            }
          }
        }
      }
    }
    template {
      data        = "{{ key \"${config_key}\" }}"
      destination = "local/config.yml"
    }
    task "dest_${dest_name}" {
      driver = "docker"
      config {
        image = "${dests[dest_name].nomad.image}"
        args  = ["-dest", "-name", "${dest_name}"]
        volumes = [
          "local/config.yml:/etc/mysqlsync/config.yml"
        ]
      }
    }
  }
%{ endfor ~}
}

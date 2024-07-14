job "mysqlsync-${repl.name}" {
  region = "dc1"
  datacenters = ${jsonencode(all_datacenters)}
  group "repl-${repl.name}" {
    constraint {
      attribute = "$${node.datacenter}"
      value     = "${repl.nomad.datacenter}"
    }
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
    task "repl-${repl.name}" {
      driver = "docker"
      template {
        data        = "{{ key \"${config_key}\" }}"
        destination = "local/config.yml"
      }
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
    constraint {
      attribute = "$${node.datacenter}"
      value     = "${dests[dest_name].nomad.datacenter}"
    }
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
    task "dest_${dest_name}" {
      template {
        data        = "{{ key \"${config_key}\" }}"
        destination = "local/config.yml"
      }
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

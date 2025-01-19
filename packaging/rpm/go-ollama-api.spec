Name:           go-ollama-api
Version:        %{_version}
Release:        1%{?dist}
Summary:        Go-based API proxy for Ollama with API key management

License:        MIT
URL:            https://github.com/erock530/go-ollama-api
Source0:        %{name}-%{version}.tar.gz

BuildRequires:  golang >= 1.21
BuildRequires:  sqlite-devel
Requires:       sqlite

%description
A Go-based API proxy for Ollama with API key management, rate limiting, and webhook support.

%prep
%setup -q

%build
go build -ldflags="-X main.Version=%{version} -X main.CommitHash=%{_commit_hash} -X main.BuildTime=%{_build_time}" -o %{name} ./cmd/server

%install
rm -rf $RPM_BUILD_ROOT
install -d %{buildroot}%{_bindir}
install -d %{buildroot}%{_unitdir}
install -d %{buildroot}%{_sysconfdir}/%{name}
install -p -m 755 %{name} %{buildroot}%{_bindir}/%{name}
install -p -m 644 packaging/systemd/%{name}.service %{buildroot}%{_unitdir}/%{name}.service

%pre
getent group go-ollama-api >/dev/null || groupadd -r go-ollama-api
getent passwd go-ollama-api >/dev/null || \
    useradd -r -g go-ollama-api -d /var/lib/%{name} -s /sbin/nologin \
    -c "Go Ollama API Server" go-ollama-api
exit 0

%post
%systemd_post %{name}.service

%preun
%systemd_preun %{name}.service

%postun
%systemd_postun_with_restart %{name}.service

%files
%{_bindir}/%{name}
%{_unitdir}/%{name}.service
%dir %{_sysconfdir}/%{name}
%doc README.md

%changelog
* Wed Feb 21 2024 Eric <erock530@github.com> - 1.0.0-1
- Initial package release

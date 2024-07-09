#!/usr/bin/env bash
PATH=/bin:/sbin:/usr/bin:/usr/sbin:/usr/local/bin:/usr/local/sbin:~/bin
export PATH

init_var() {
  ECHO_TYPE="echo -e"

  HUI_DATA_DOCKER="/h-ui/"
  HUI_DATA_MANUAL="/usr/local/h-ui/"

  h_ui_port=8081
  h_ui_time_zone=Asia/Shanghai
}

echo_content() {
  case $1 in
  "red")
    ${ECHO_TYPE} "\033[31m$2\033[0m"
    ;;
  "green")
    ${ECHO_TYPE} "\033[32m$2\033[0m"
    ;;
  "yellow")
    ${ECHO_TYPE} "\033[33m$2\033[0m"
    ;;
  "blue")
    ${ECHO_TYPE} "\033[34m$2\033[0m"
    ;;
  "purple")
    ${ECHO_TYPE} "\033[35m$2\033[0m"
    ;;
  "skyBlue")
    ${ECHO_TYPE} "\033[36m$2\033[0m"
    ;;
  "white")
    ${ECHO_TYPE} "\033[37m$2\033[0m"
    ;;
  esac
}

check_sys() {
  if [[ $(id -u) != "0" ]]; then
    echo_content red "You must be root to run this script"
    exit 1
  fi

  if [[ $(command -v yum) ]]; then
    package_manager='yum'
  elif [[ $(command -v dnf) ]]; then
    package_manager='dnf'
  elif [[ $(command -v apt-get) ]]; then
    package_manager='apt-get'
  elif [[ $(command -v apt) ]]; then
    package_manager='apt'
  fi

  if [[ -z "${package_manager}" ]]; then
    echo_content red "This system is not currently supported"
    exit 1
  fi

  if [[ -n $(find /etc -name "redhat-release") ]] || grep </proc/version -q -i "centos"; then
    release="centos"
  elif grep </etc/issue -q -i "debian" && [[ -f "/etc/issue" ]] || grep </etc/issue -q -i "debian" && [[ -f "/proc/version" ]]; then
    release="debian"
  elif grep </etc/issue -q -i "ubuntu" && [[ -f "/etc/issue" ]] || grep </etc/issue -q -i "ubuntu" && [[ -f "/proc/version" ]]; then
    release="ubuntu"
  fi

  if [[ -z "${release}" ]]; then
    echo_content red "Only supports CentOS 7+/Ubuntu 18+/Debian 10+"
    exit 1
  fi

  if [[ $(arch) =~ ("x86_64"|"amd64"|"arm64"|"aarch64"|"ppc64le") ]]; then
    get_arch=$(arch)
  fi

  if [[ -z "${get_arch}" ]]; then
    echo_content red "Only supports x86_64/amd64 arm64/aarch64 ppc64le"
    exit 1
  fi
}

install_depend() {
  if [[ "${package_manager}" == 'apt-get' || "${package_manager}" == 'apt' ]]; then
    ${package_manager} update -y
  fi
  ${package_manager} install -y \
    curl \
    wget
}

setup_docker() {
  mkdir -p /etc/docker
  cat >/etc/docker/daemon.json <<EOF
{
  "log-driver":"json-file",
  "log-opts":{
      "max-size":"100m"
  }
}
EOF
  systemctl daemon-reload
}

install_docker() {
  if [[ ! $(command -v docker) ]]; then
    echo_content green "---> Install Docker"

    if [[ "${release}" == "centos" ]]; then
      ${package_manager} install -y yum-utils
      ${package_manager}-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
      ${package_manager} makecache || ${package_manager} makecache fast
    elif [[ "${release}" == "debian" || "${release}" == "ubuntu" ]]; then
      ${package_manager} update -y
      ${package_manager} install -y install ca-certificates
      install -m 0755 -d /etc/apt/keyrings
      curl -fsSL https://download.docker.com/linux/${release}/gpg -o /etc/apt/keyrings/docker.asc
      chmod a+r /etc/apt/keyrings/docker.asc
      echo \
        "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/${release} \
                    $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list >/dev/null
      tee /etc/apt/sources.list.d/docker.list >/dev/null
      ${package_manager} update -y
    fi

    ${package_manager} install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin

    setup_docker

    systemctl enable docker && systemctl restart docker

    if [[ $(command -v docker) ]]; then
      echo_content skyBlue "---> Docker install successful"
    else
      echo_content red "---> Docker install failed"
      exit 1
    fi
  else
    echo_content skyBlue "---> You have installed Docker"
  fi
}

install_h_ui_docker() {
  mkdir -p ${HUI_DATA_DOCKER}

  read -r -p "Please enter the port of H UI (default: 8081): " h_ui_port
  [[ -z "${h_ui_port}" ]] && h_ui_port="8081"

  read -r -p "Please enter the Time zone of H UI (default: Asia/Shanghai): " h_ui_time_zone
  [[ -z "${h_ui_time_zone}" ]] && h_ui_time_zone="Asia/Shanghai"

  docker pull jonssonyan/h-ui &&
  docker run -d --name h-ui --restart always \
    --network=host \
    -e TZ=${h_ui_time_zone} \
    -v /h-ui/bin:/h-ui/bin \
    -v /h-ui/data:/h-ui/data \
    -v /h-ui/export:/h-ui/export \
    -v /h-ui/logs:/h-ui/logs \
    jonssonyan/h-ui \
    ./h-ui -p ${h_ui_port}
}

upgrade_h_ui_docker() {
  if [[ ! $(command -v docker) ]]; then
   echo_content red "---> Docker not installed"
   exit 0
  fi
  if [[ -z $(docker ps -a -q -f "name=^h-ui$") ]]; then
   echo_content red "---> H UI not installed"
   exit 0
  fi
  docker stop h-ui &&
  curl -fsSL https://github.com/jonssonyan/h-ui/releases/latest/download/h-ui-linux-amd64 -o /h-ui/h-ui && docker exec h-ui chmod +x /h-ui/h-ui &&
  docker restart h-ui
  echo_content skyBlue "---> H UI upgrade successful"
}

uninstall_h_ui_docker() {
  if [[ ! $(command -v docker) ]]; then
    echo_content red "---> Docker not installed"
    exit 0
  fi
  if [[ -z $(docker ps -a -q -f "name=^h-ui$") ]]; then
    echo_content red "---> H UI not installed"
    exit 0
  fi
  docker rm -f h-ui &&
  docker rmi jonssonyan/h-ui &&
  rm -rf /h-ui/
}

install_h_ui_manual() {
  mkdir -p ${HUI_DATA_MANUAL}

  curl -fsSL https://github.com/jonssonyan/h-ui/releases/latest/download/h-ui-linux-amd64 -o /usr/local/h-ui/h-ui && chmod +x /usr/local/h-ui/h-ui &&
  curl -fsSL https://raw.githubusercontent.com/jonssonyan/h-ui/main/h-ui.service -o /etc/systemd/system/h-ui.service &&
  systemctl daemon-reload &&
  systemctl enable h-ui &&
  systemctl restart h-ui
}

upgrade_h_ui_manual() {
  systemctl stop h-ui &&
  curl -fsSL https://github.com/jonssonyan/h-ui/releases/latest/download/h-ui-linux-amd64 -o /usr/local/h-ui/h-ui && chmod +x /usr/local/h-ui/h-ui &&
  systemctl restart h-ui
}

uninstall_h_ui_manual() {
  systemctl stop h-ui &&
  rm -rf /etc/systemd/system/h-ui.service /usr/local/h-ui/
}

main() {
  cd "$HOME" || exit 0
  init_var
  check_sys
  install_depend
  clear
  echo_content red "\n=============================================================="
  echo_content skyBlue "Recommended OS: CentOS 8+/Ubuntu 20+/Debian 11+"
  echo_content skyBlue "Description: Quick Installation of H UI"
  echo_content skyBlue "Author: jonssonyan <https://jonssonyan.com>"
  echo_content skyBlue "Github: https://github.com/jonssonyan/h-ui"
  echo_content red "\n=============================================================="
  echo_content yellow "1. Install H UI (Docker)"
  echo_content yellow "2. Upgrade H UI (Docker)"
  echo_content yellow "3. Uninstall H UI (Docker)"
  echo_content red "\n=============================================================="
  echo_content yellow "4. Install H UI (Manual)"
  echo_content yellow "5. Upgrade H UI (Manual)"
  echo_content yellow "6. Uninstall H UI (Manual)"
  read -r -p "Please choose: " input_option
  case ${input_option} in
  1)
    install_docker
    install_h_ui_docker
    ;;
  2)
    upgrade_h_ui_docker
    ;;
  3)
    uninstall_h_ui_docker
    ;;
  4)
    install_h_ui_manual
    ;;
  5)
    upgrade_h_ui_manual
    ;;
  6)
    uninstall_h_ui_manual
    ;;
  *)
    echo_content red "No such option"
    ;;
  esac
}

main

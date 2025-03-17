#include <asl.h>
#include <fcntl.h>
#include <stdio.h>
#include <string.h>
#include <time.h>

#include "asl_darwin.h"

// logs
static aslclient asl = NULL;
static aslmsg log_msg = NULL;

// asl is deprecated in favor of os_log starting with macOS 10.12.
#pragma GCC diagnostic ignored "-Wdeprecated-declarations"

void apple_asl_logger_log(int level, const char *message) {
  if (!asl) {
    asl = asl_open("Docker", "com.docker.docker", 0);
    log_msg = asl_new(ASL_TYPE_MSG);
  }

  // The max length for log entries is 1024 bytes.  Beyond, they are
  // truncated.  In that case, split into several log entries.
  const size_t len = strlen(message);
  if (len < 1024)
    asl_log(asl, log_msg, level, "%s", message);
  else {
    enum { step = 1000 };
    for (int pos = 0; pos < len; pos += step) {
      asl_log(asl, log_msg, level,
              "%s%.*s%s",
              pos ? "[...] " : "",
              step, message + pos,
              pos + step < len ? " [...]" : "");
    }
  }
}

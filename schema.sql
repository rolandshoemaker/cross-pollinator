CREATE TABLE `validRoots` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `dn` varchar(255) NOT NULL,
  `logID` binary(32) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `dn_logID` (`dn`, `logID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `logEntries` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `hash` binary(32) NOT NULL,
  `rootDN` varchar(255) NOT NULL,
  `entryNum` bigint(20) NOT NULL,
  `logID` binary(32) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `entryNum_logID` (`entryNum`, `logID`)
  UNIQUE KEY `hash_logID` (`hash`, `logID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `log` (
  `id` binary(32) NOT NULL,
  `name` varchar(255) NOT NULL,
  `currentIndex` bigint(20) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `progress` (
  `srcLog` binary(32) NOT NULL,
  `dstLog` binary(32) NOT NULL,
  `currentIndex` bigint(20) NOT NULL,
  UNIQUE KEY `src_dst` (`srcLog`, `dstLog`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

#ifndef HCONFIG_H_
#define HCONFIG_H_

/* #undef FAST_BUILD */
/* #undef SCIP_DEV */
/* #undef HiGHSDEV */
/* #undef OSI_FOUND */
/* #undef ZLIB_FOUND */
#define CMAKE_BUILD_TYPE "Release"
/* #undef HiGHSRELEASE */
/* #undef HIGHSINT64 */
#define HIGHS_HAVE_MM_PAUSE
#define HIGHS_HAVE_BUILTIN_CLZ
/* #undef HIGHS_HAVE_BITSCAN_REVERSE */

#define HIGHS_GITHASH "14c0f2256"
#define HIGHS_COMPILATION_DATE "2022-11-01"
#define HIGHS_VERSION_MAJOR 1
#define HIGHS_VERSION_MINOR 3
#define HIGHS_VERSION_PATCH 1
#define HIGHS_DIR "/app/highs"

#endif /* HCONFIG_H_ */


// This file was generated automatically by "glapi2go" and contains declarations
// of public functions from "glapli.c".

#ifndef _glapi_h_
#define _glapi_h_

#include "glcorearb.h"

// Loads the OpenGL function pointers
int glapiLoad(void);

// Set the internal flag to enable/disable OpenGL error checking
void glapiCheckError(int check);

// Program binaries (moonBASIC patch)
int mbBinarySupported(void);
void mbProgramRetrievable(GLuint program);
void mbProgramBinary(GLuint program, GLenum format, const void *data, GLsizei length);
GLsizei mbGetProgramBinary(GLuint program, GLsizei size, GLenum *format, void *data);

#endif

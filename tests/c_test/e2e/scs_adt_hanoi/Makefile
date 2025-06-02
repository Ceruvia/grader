TARGET = prog
LIBS = -lm
CC = gcc
CFLAGS = -std=c99 -g -Wall -D_INC_STRING

.PHONY: compile clean
.PRECIOUS: $(TARGET) $(OBJECTS)

OBJECTS = $(patsubst %.c, %.o, $(wildcard *.c))
HEADERS = $(wildcard *.h)

# Ensure include.c exists before compiling, if INPUT_FILE is provided
compile: $(TARGET)
	@if [ -n "$(INPUT_FILE)" ]; then \
		cp "$(INPUT_FILE)" include.c; \
	fi
	@if [ -f include.c ]; then \
		echo "Using include.c"; \
	else \
		echo ""; \
	fi

# target: prerequisites
#    recipes
# compile without linking them, will produce a .o file
%.o: %.c $(HEADERS)
	$(CC) $(CFLAGS) -c $< -o $@

# link all object files, will produce an executable file named TARGET
$(TARGET): $(OBJECTS)
	$(CC) $(OBJECTS) -Wall $(LIBS) -o $@

# remove all .o files and executable TARGET
clean:
	-rm -f *.o
	-rm -f $(TARGET)

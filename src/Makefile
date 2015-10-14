all: singularity

singularity:
	gcc -g -o singularity singularity.c
	chown root:root singularity
	chmod +s singularity

clean:
	rm -f singularity

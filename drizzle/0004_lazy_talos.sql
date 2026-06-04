CREATE TABLE "solved_problems" (
	"user_id" integer NOT NULL,
	"problem_id" integer NOT NULL,
	"first_submission_id" integer NOT NULL,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL,
	CONSTRAINT "solved_problems_user_id_problem_id_pk" PRIMARY KEY("user_id","problem_id")
);
--> statement-breakpoint
CREATE INDEX "solved_problems_problem_idx" ON "solved_problems" USING btree ("problem_id");
import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import LoginForm from "./LoginForm";
import { clearSession, getAccessToken, isAuthenticated } from "./auth/session";

// LoginForm imports { login } from "./api/auth" — mock that module so the
// test drives the real component + the real session module (proving the
// actual login->setSession wiring works) without hitting the network.
vi.mock("./api/auth", () => ({ login: vi.fn() }));
import { login } from "./api/auth";

beforeEach(() => {
  clearSession();
  vi.clearAllMocks();
});

describe("LoginForm", () => {
  it("renders email, password, and a submit control", () => {
    render(<LoginForm onShowSignup={() => {}} />);
    expect(screen.getByLabelText(/email/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/^password$/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /sign in/i })).toBeInTheDocument();
  });

  it("submitting valid credentials logs the session in", async () => {
    login.mockResolvedValue({ accessToken: "access-1", refreshToken: "refresh-1" });
    const user = userEvent.setup();
    render(<LoginForm onShowSignup={() => {}} />);

    await user.type(screen.getByLabelText(/email/i), "admin@campaigntracker.dev");
    await user.type(screen.getByLabelText(/^password$/i), "ChangeMe123!admin");
    await user.click(screen.getByRole("button", { name: /sign in/i }));

    await waitFor(() => expect(isAuthenticated()).toBe(true));
    expect(getAccessToken()).toBe("access-1");
    expect(login).toHaveBeenCalledWith("admin@campaigntracker.dev", "ChangeMe123!admin");
  });

  it("shows the server's error message and does not log the session in on failure", async () => {
    login.mockRejectedValue(new Error("invalid email or password"));
    const user = userEvent.setup();
    render(<LoginForm onShowSignup={() => {}} />);

    await user.type(screen.getByLabelText(/email/i), "admin@campaigntracker.dev");
    await user.type(screen.getByLabelText(/^password$/i), "wrong");
    await user.click(screen.getByRole("button", { name: /sign in/i }));

    expect(await screen.findByText(/invalid email or password/i)).toBeInTheDocument();
    expect(isAuthenticated()).toBe(false);
  });

  it("clicking \"Sign up\" calls onShowSignup", async () => {
    const onShowSignup = vi.fn();
    const user = userEvent.setup();
    render(<LoginForm onShowSignup={onShowSignup} />);

    await user.click(screen.getByRole("button", { name: /sign up/i }));
    expect(onShowSignup).toHaveBeenCalledTimes(1);
  });
});
